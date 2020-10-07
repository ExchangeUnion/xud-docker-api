package connext

import (
	"errors"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud"
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
		svc, err := t.GetServiceManager().GetService("xud")
		if err == nil {
			xudSvc := svc.(*xud.XudService)
			info, err := xudSvc.GetInfo()
			if err == nil {
				return info.Connext.Status, nil
			}
		}

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
	value, err := t.Getenv("CONNEXT_ETH_PROVIDER_URL")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("CONNEXT_ETH_PROVIDER_URL not found")
	}
	return value, nil
}
