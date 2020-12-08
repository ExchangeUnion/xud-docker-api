package litecoind

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/bitcoind"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
	docker "github.com/docker/docker/client"
)

type Service struct {
	*bitcoind.Service
}

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	l2ServiceName string,
	rpcConfig config.RpcConfig,
) *Service {
	return &Service{
		bitcoind.New(name, services, containerName, dockerClient, l2ServiceName, rpcConfig),
	}
}
