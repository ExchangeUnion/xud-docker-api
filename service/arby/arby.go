package arby

import (
	"context"
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
	if status == "Container running" {
		return "Ready"
	} else {
		return status
	}
}

func (t *Service) Close() error {
	_ = t.RpcClient.Close()
	return nil
}
