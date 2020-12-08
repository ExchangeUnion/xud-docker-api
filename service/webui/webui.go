package webui

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
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		return "Ready", nil
	} else {
		return status, nil
	}
}

func (t *Service) Close() error {
	return nil
}
