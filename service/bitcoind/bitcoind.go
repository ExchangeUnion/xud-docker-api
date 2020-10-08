package bitcoind

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/lnd"
	"github.com/ybbus/jsonrpc"
)

type BitcoindService struct {
	*service.SingleContainerService
	rpcOptions    *service.RpcOptions
	rpcClient     jsonrpc.RPCClient
	l2ServiceName string
}

type Mode string

const (
	Native   Mode = "native"
	External Mode = "external"
	Light    Mode = "light"
	Unknown  Mode = "unknown"
)

func New(
	name string,
	containerName string,
	l2ServiceName string,
) *BitcoindService {
	return &BitcoindService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
		l2ServiceName:          l2ServiceName,
	}
}

func (t *BitcoindService) ConfigureRpc(options *service.RpcOptions) {
	t.rpcOptions = options
}

func (t *BitcoindService) getRpcClient() jsonrpc.RPCClient {
	if t.rpcClient == nil {
		addr := fmt.Sprintf("http://%s:%d", t.rpcOptions.Host, t.rpcOptions.Port)
		t.rpcClient = jsonrpc.NewClientWithOpts(addr, &jsonrpc.RPCClientOpts{
			CustomHeaders: map[string]string{
				"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("xu"+":"+"xu")),
			},
		})
	}
	return t.rpcClient
}

func (t *BitcoindService) GetBlockchainInfo() (*jsonrpc.RPCResponse, error) {
	client := t.getRpcClient()
	response, err := client.Call("getblockchaininfo")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (t *BitcoindService) getL2Service() (*lnd.LndService, error) {
	s, err := t.GetServiceManager().GetService(t.l2ServiceName)
	if err != nil {
		return nil, err
	}
	lndSvc, ok := s.(*lnd.LndService)
	if !ok {
		return nil, errors.New("cannot convert to LndService")
	}
	return lndSvc, nil
}

func (t *BitcoindService) getMode() (Mode, error) {
	lndSvc, err := t.getL2Service()
	if err != nil {
		return Unknown, err
	}
	backend, err := lndSvc.GetBackendNode()
	if err != nil {
		return Unknown, err
	}
	if backend == "bitcoind" || backend == "litecoind" {
		// could be native or external
		values, err := lndSvc.GetConfigValues(fmt.Sprintf("%s.rpchost", backend))
		if err != nil {
			return Unknown, err
		}
		host := values[0]
		if host == backend {
			return Native, nil
		} else {
			return External, nil
		}
	} else if backend == "neutrino" {
		return Light, nil
	}
	return Unknown, nil
}

func (t *BitcoindService) GetStatus() (string, error) {
	mode, err := t.getMode()
	if err != nil {
		return "", err
	}
	switch mode {
	case Native:
		status, err := t.SingleContainerService.GetStatus()
		if status != "Container running" {
			return status, nil
		}
		resp, err := t.GetBlockchainInfo()
		if err != nil {
			return fmt.Sprintf("Waiting for %s to come up...", t.GetName()), nil
		}
		if resp.Error != nil {
			// Loading block index...
			return resp.Error.Message, nil
		}
		r := resp.Result.(map[string]interface{})
		current, err := r["blocks"].(json.Number).Int64()
		if err != nil {
			return "", err
		}
		total, err := r["headers"].(json.Number).Int64()
		if err != nil {
			return "", err
		}
		if current > 0 && current == total {
			return "Ready", nil
		} else {
			if total == 0 {
				return "Syncing 0.00% (0/0)", nil
			} else {
				p := float32(current) / float32(total) * 100.0
				return fmt.Sprintf("Syncing %.2f%% (%d/%d)", p, current, total), nil
			}
		}
	case External:
		// TODO Unavailable (connection to external failed)
		return "Ready (connected to external)", nil
	case Light:
		return "Ready (light mode)", nil
	default:
		return "Error: Unknown mode", nil
	}
}
