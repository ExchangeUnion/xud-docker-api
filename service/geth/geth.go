package geth

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/ybbus/jsonrpc"
	"log"
	"strconv"
	"strings"
)

type GethService struct {
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
	rpcClient  jsonrpc.RPCClient
}

type Mode string

const (
	Native   Mode = "native"
	External Mode = "external"
	Infura   Mode = "infura"
	Light    Mode = "light"
	Unknown  Mode = "unknown"
)

func New(name string, containerName string) *GethService {
	return &GethService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
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

	log.Printf("syncing is %v", syncing)

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

func (t *GethService) checkEthRpc(url string) bool {
	return true
}

func (t *GethService) getMode() Mode {
	// TODO get geth mode
	return Unknown
}

func (t *GethService) GetStatus() (string, error) {
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
