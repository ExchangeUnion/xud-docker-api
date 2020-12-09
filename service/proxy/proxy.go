package proxy

import (
	"context"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	docker "github.com/docker/docker/client"
)

type Service struct {
	*core.SingleContainerService
}

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
) *Service {
	return &Service{
		SingleContainerService: core.NewSingleContainerService(name, services, containerName, dockerClient),
	}
}

func (t *Service) GetStatus(ctx context.Context) string {
	return "Ready"
}

func (t *Service) Close() error {
	return nil
}
