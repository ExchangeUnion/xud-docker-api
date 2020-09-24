package bitcoind

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"testing"
)

func Test1(t *testing.T) {
	options := service.RpcOptions{
		Host: "192.168.11.231",
		Port: 8332,
		Credential: service.UsernamePasswordCredential{
			Username: "xu",
			Password: "xu",
		},
	}

	s := New("bitcoind", "testnet_bitcoind_1", "lndbtc")
	s.ConfigureRpc(&options)

	info, err := s.GetBlockchainInfo()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", info)
}