package geth

import (
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/connext"
	"github.com/ybbus/jsonrpc"
	"strconv"
	"strings"
)

type GethService struct {
	*service.SingleContainerService
	rpcOptions     *service.RpcOptions
	rpcClient      jsonrpc.RPCClient
	l2ServiceName  string
	lightProviders []string
}

type Mode string

const (
	Native   Mode = "native"
	External Mode = "external"
	Infura   Mode = "infura"
	Light    Mode = "light"
	Unknown  Mode = "unknown"
)

func New(name string, containerName string, l2ServiceName string, lightProviders []string) *GethService {
	return &GethService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
		l2ServiceName:          l2ServiceName,
		lightProviders:         lightProviders,
	}
}

func (t *GethService) ConfigureRpc(options *service.RpcOptions) {
	t.rpcOptions = options
}

func (t *GethService) getRpcClient() jsonrpc.RPCClient {
	if t.rpcClient == nil {
		addr := fmt.Sprintf("http://%s:%d", t.rpcOptions.Host, t.rpcOptions.Port)
		t.rpcClient = jsonrpc.NewClientWithOpts(addr, &jsonrpc.RPCClientOpts{})
	}
	return t.rpcClient
}

type Syncing struct {
	CurrentBlock  int64
	HighestBlock  int64
	KnownStates   int64
	PulledStates  int64
	StartingBlock int64
}

func parseHex(value string) (int64, error) {
	value = strings.Replace(value, "0x", "", 1)
	i64, err := strconv.ParseInt(value, 16, 32)
	if err != nil {
		return 0, err
	}
	return i64, nil
}

func (t *GethService) EthSyncing() (*Syncing, error) {
	result, err := t.getRpcClient().Call("eth_syncing")
	if err != nil {
		return nil, err
	}

	var syncing map[string]string
	err = result.GetObject(&syncing)
	if err != nil {
		_, err := result.GetBool()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	currentBlock, err := parseHex(syncing["currentBlock"])
	if err != nil {
		return nil, err
	}

	highestBlock, err := parseHex(syncing["highestBlock"])
	if err != nil {
		return nil, err
	}

	knownStates, err := parseHex(syncing["knownStates"])
	if err != nil {
		return nil, err
	}

	pulledStates, err := parseHex(syncing["pulledStates"])
	if err != nil {
		return nil, err
	}

	startingBlock, err := parseHex(syncing["startingBlock"])
	if err != nil {
		return nil, err
	}

	return &Syncing{
		CurrentBlock:  currentBlock,
		HighestBlock:  highestBlock,
		KnownStates:   knownStates,
		PulledStates:  pulledStates,
		StartingBlock: startingBlock,
	}, nil
}

func (t *GethService) EthBlockNumber() (int64, error) {
	result, err := t.getRpcClient().Call("eth_blockNumber")
	if err != nil {
		return 0, err
	}
	s, err := result.GetString()
	if err != nil {
		return 0, err
	}
	blockNumber, err := parseHex(s)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func explainNetVersion(version string) string {
	switch version {
	case "1":
		return "Mainnet"
	case "2":
		return "Testnet (Morden, deprecated!)"
	case "3":
		return "Testnet (Ropsten)"
	case "4":
		return "Testnet (Rinkeby)"
	case "42":
		return "Testnet (Kovan)"
	default:
		return version
	}
}

func (t *GethService) checkEthRpc(url string) bool {
	client := jsonrpc.NewClientWithOpts(url, &jsonrpc.RPCClientOpts{})
	result, err := client.Call("net_version")
	if err != nil {
		return false
	}
	version, err := result.GetString()
	if err != nil {
		return false
	}
	t.GetLogger().Infof("Ethereum provider %s net_version is %s", url, explainNetVersion(version))
	return true
}

func (t *GethService) getL2Service() (*connext.ConnextService, error) {
	s, err := t.GetServiceManager().GetService(t.l2ServiceName)
	if err != nil {
		return nil, err
	}
	connextSvc, ok := s.(*connext.ConnextService)
	if !ok {
		return nil, errors.New("cannot convert to ConnextService")
	}
	return connextSvc, nil
}

func (t *GethService) isLightProvider(provider string) bool {
	for _, item := range t.lightProviders {
		if item == provider {
			return true
		}
	}
	return false
}

func (t *GethService) getProvider() (string, error) {
	connextSvc, err := t.getL2Service()
	if err != nil {
		return "", err
	}

	provider, err := connextSvc.GetEthProvider()
	if err != nil {
		return "", err
	}

	return provider, nil
}

func (t *GethService) getMode() (Mode, error) {
	provider, err := t.getProvider()
	if err != nil {
		return Unknown, err
	}

	if provider == "http://geth:8545" {
		return Native, nil
	} else if strings.Contains(provider, "infura") {
		return Infura, nil
	} else if t.isLightProvider(provider) {
		return Light, nil
	} else {
		return External, nil
	}
}

func (t *GethService) getExternalStatus() (string, error) {
	provider, err := t.getProvider()
	if err != nil {
		return "No provider", err
	}
	if t.checkEthRpc(provider) {
		return "Ready (connected to external)", nil
	} else {
		return "Unavailable (connection to external failed)", nil
	}
}

func (t *GethService) getInfuraStatus() (string, error) {
	provider, err := t.getProvider()
	if err != nil {
		return "No provider", err
	}
	if t.checkEthRpc(provider) {
		return "Ready (connected to Infura)", nil
	} else {
		return "Unavailable (connection to Infura failed)", nil
	}
}

func (t *GethService) getLightStatus() (string, error) {
	provider, err := t.getProvider()
	if err != nil {
		return "No provider", err
	}
	if t.checkEthRpc(provider) {
		return "Ready (light mode)", nil
	} else {
		return "Unavailable (light mode failed)", nil
	}
}

func (t *GethService) GetStatus() (string, error) {
	mode, err := t.getMode()
	if err != nil {
		return "", err
	}

	if mode == External {
		return t.getExternalStatus()
	} else if mode == Infura {
		return t.getInfuraStatus()
	} else if mode == Light {
		return t.getLightStatus()
	}

	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		syncing, err := t.EthSyncing()
		if err != nil {
			return "Waiting for geth to come up...", err
		}
		if syncing != nil {
			current := syncing.CurrentBlock
			total := syncing.HighestBlock
			p := float32(current) / float32(total) * 100.0
			return fmt.Sprintf("Syncing %.2f%% (%d/%d)", p, current, total), nil
		} else {
			blockNumber, err := t.EthBlockNumber()
			if err != nil {
				return "Waiting for geth to come up...", err
			}
			if blockNumber == 0 {
				return "Waiting for sync", nil
			} else {
				return "Ready", nil
			}
		}
	} else {
		return status, nil
	}
}
