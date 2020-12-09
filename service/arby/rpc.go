package arby

import (
	"github.com/ExchangeUnion/xud-docker-api/config"
)

type RpcClient struct {
}

func NewRpcClient(config config.RpcConfig) *RpcClient {
	return &RpcClient{}
}

func (t *RpcClient) Close() error {
	return nil
}
