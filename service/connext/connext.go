package connext

import (
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"net/http"
	"strings"
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
	container, err := t.GetContainer()
	if err != nil {
		return "", err
	}
	key := "CONNEXT_ETH_PROVIDER_URL"
	prefix := key + "="
	for _, env := range container.Config.Env {
		if strings.HasPrefix(env, prefix) {
			url := strings.Replace(env, prefix, "", 1)
			return url, nil
		}
	}
	return "", errors.New(fmt.Sprintf("%s not found", key))
}
