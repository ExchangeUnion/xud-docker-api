package proxy

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
)

type ProxyService struct {
	*service.SingleContainerService
}

func New(
	name string,
	containerName string,
) *ProxyService {
	return &ProxyService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *ProxyService) GetStatus() (string, error) {
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
