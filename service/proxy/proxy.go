package proxy

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
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

func (t *Service) GetStatus() (string, error) {
	_, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	return "Ready", nil
}

func (t *Service) Close() error {
	return nil
}
