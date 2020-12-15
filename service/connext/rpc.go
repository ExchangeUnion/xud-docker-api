package connext

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/sirupsen/logrus"
	"net/http"
)

type RpcClient struct {
	url            string
	healthEndpoint string
	client         *http.Client

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	url := fmt.Sprintf("http://%s:%d", host, port)
	return &RpcClient{
		url:            url,
		healthEndpoint: fmt.Sprintf("%s/health", url),
		client:         &http.Client{},
		logger:         service.GetLogger().WithField("scope", "RPC"),
		service:        service,
	}
}

func (t *RpcClient) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.healthEndpoint, nil)
	if err != nil {
		t.logger.Errorf("Failed to create HTTP request: %s", err)
		return false
	}
	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Errorf("Failed to send HTTP request: %s", err)
		return false
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.logger.Errorf("Faield to close HTTP response body: %s", err)
		}
	}()
	if resp.StatusCode == http.StatusNoContent {
		return true
	}
	return false
}

func (t *RpcClient) Close() error {
	return nil
}
