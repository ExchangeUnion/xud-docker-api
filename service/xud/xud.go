package xud

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
	docker "github.com/docker/docker/client"
	"strings"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient
}

type XudRpc struct {
	Host string
	Port int
	Cert string
}

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	rpcConfig config.RpcConfig,
) *Service {
	return &Service{
		SingleContainerService: core.NewSingleContainerService(name, services, containerName, dockerClient),
		RpcClient:              NewRpcClient(rpcConfig),
	}
}

func (t *Service) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		resp, err := t.GetInfo()
		if err != nil {
			if strings.Contains(err.Error(), "xud is locked") {
				return "Wallet locked. Unlock with xucli unlock.", nil
			} else if strings.Contains(err.Error(), "no such file or directory, open '/root/.xud/tls.cert'") {
				return "Starting...", nil
			} else if strings.Contains(err.Error(), "xud is starting") {
				return "Starting...", nil
			}
			return "", err
		}
		lndbtcStatus := resp.Lnd["BTC"].Status
		lndltcStatus := resp.Lnd["LTC"].Status
		connextStatus := resp.Connext.Status

		if lndbtcStatus == "Ready" && lndltcStatus == "Ready" && connextStatus == "Ready" {
			return "Ready", nil
		}

		if strings.Contains(lndbtcStatus, "has no active channels") ||
			strings.Contains(lndltcStatus, "has no active channels") ||
			strings.Contains(connextStatus, "has no active channels") {
			return "Waiting for channels", nil
		}

		var notReady []string
		if lndbtcStatus != "Ready" {
			notReady = append(notReady, "lndbtc")
		}
		if lndltcStatus != "Ready" {
			notReady = append(notReady, "lndltc")
		}
		if connextStatus != "Ready" {
			notReady = append(notReady, "connext")
		}

		return "Waiting for " + strings.Join(notReady, ", "), nil
	} else {
		return status, nil
	}
}

func (t *Service) Close() error {
	_ = t.RpcClient.Close()
	return nil
}

