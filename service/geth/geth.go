package geth

import (
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/connext"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
	docker "github.com/docker/docker/client"
	"github.com/ybbus/jsonrpc"
	"strings"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient

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

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	l2ServiceName string,
	lightProviders []string,
	rpcConfig config.RpcConfig,
) *Service {
	return &Service{
		SingleContainerService: core.NewSingleContainerService(name, services, containerName, dockerClient),
		RpcClient:              NewRpcClient(rpcConfig),
		l2ServiceName:          l2ServiceName,
		lightProviders:         lightProviders,
	}
}

func (t *Service) checkEthRpc(url string) bool {
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

func (t *Service) getL2Service() (*connext.Service, error) {
	s := t.GetService(t.l2ServiceName)
	connextSvc, ok := s.(*connext.Service)
	if !ok {
		return nil, errors.New("cannot convert to ConnextService")
	}
	return connextSvc, nil
}

func (t *Service) isLightProvider(provider string) bool {
	for _, item := range t.lightProviders {
		if item == provider {
			return true
		}
	}
	return false
}

func (t *Service) getProvider() (string, error) {
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

func (t *Service) getMode() (Mode, error) {
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

func (t *Service) getExternalStatus() (string, error) {
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

func (t *Service) getInfuraStatus() (string, error) {
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

func (t *Service) getLightStatus() (string, error) {
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

func (t *Service) GetStatus() (string, error) {
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

func (t *Service) Close() error {
	_ = t.RpcClient.Close()
	return nil
}

