package connext

import (
	"context"
	"errors"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	docker "github.com/docker/docker/client"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient
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

func (t *Service) GetStatus(ctx context.Context) string {
	status := t.SingleContainerService.GetStatus(ctx)
	if status != "Container running" {
		return status
	}

	// container is running

	//svc := t.GetService("xud")
	//if svc != nil {
	//	xudSvc := svc.(*xud.Service)
	//	info, err := xudSvc.GetInfo()
	//	if err == nil {
	//		return info.Connext.Status
	//	}
	//}

	if t.IsHealthy() {
		return "Ready"
	} else {
		return "Starting..."
	}
}

func (t *Service) GetEthProvider() (string, error) {
	value, err := t.Getenv("CONNEXT_ETH_PROVIDER_URL")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("CONNEXT_ETH_PROVIDER_URL not found")
	}
	return value, nil
}

func (t *Service) Close() error {
	_ = t.RpcClient.Close()
	return nil
}
