package connext

import (
	"errors"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"net/http"
)

type ConnextService struct {
	*service.SingleContainerService
}

func New(name string, containerName string) *ConnextService {
	return &ConnextService{
		service.NewSingleContainerService(name, containerName),
	}
}

func (t *ConnextService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		resp, err := http.Get("http://connext:5040/health")
		if err != nil {
			return "", err
		}
		// TODO defer resp.Body.Close()
		if resp.StatusCode == http.StatusNoContent {
			return "Ready", nil
		}
		return "Starting...", nil
	} else {
		return status, nil
	}
}

func (t *ConnextService) GetEthProvider() (string, error) {
	value, err := t.GetEnvironmentVariable("CONNEXT_ETH_PROVIDER_URL")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("CONNEXT_ETH_PROVIDER_URL not found")
	}
	return value, nil
}
