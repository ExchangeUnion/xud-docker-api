package connext

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"net/http"
)

type RpcClient struct {
	url            string
	healthEndpoint string
}

func NewRpcClient(config config.RpcConfig) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	url := fmt.Sprintf("http://%s:%d", host, port)
	return &RpcClient{
		url:            url,
		healthEndpoint: fmt.Sprintf("%s/health", url),
	}
}

func (t *RpcClient) IsHealthy() bool {
	resp, err := http.Get(t.healthEndpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return true
	}
	return false
}

func (t *RpcClient) Close() error {
	return nil
}
