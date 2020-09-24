package litecoind

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/service/bitcoind"
)

type LitecoindService struct{
	*bitcoind.BitcoindService
}

func New(
	name string,
	containerName string,
	l2ServiceName string,
) *LitecoindService {
	return &LitecoindService{
		bitcoind.New(name, containerName, l2ServiceName),
	}
}