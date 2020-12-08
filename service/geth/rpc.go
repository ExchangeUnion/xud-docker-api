package geth

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ybbus/jsonrpc"
	"strconv"
	"strings"
)

type RpcClient struct {
	client jsonrpc.RPCClient
}

func NewRpcClient(config config.RpcConfig) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))

	addr := fmt.Sprintf("http://%s:%d", host, port)
	client := jsonrpc.NewClientWithOpts(addr, &jsonrpc.RPCClientOpts{})

	return &RpcClient{
		client: client,
	}
}

type Syncing struct {
	CurrentBlock  int64
	HighestBlock  int64
	KnownStates   int64
	PulledStates  int64
	StartingBlock int64
}

func parseHex(value string) (int64, error) {
	value = strings.Replace(value, "0x", "", 1)
	i64, err := strconv.ParseInt(value, 16, 32)
	if err != nil {
		return 0, err
	}
	return i64, nil
}

func (t *RpcClient) EthSyncing() (*Syncing, error) {
	result, err := t.client.Call("eth_syncing")
	if err != nil {
		return nil, err
	}

	var syncing map[string]string
	err = result.GetObject(&syncing)
	if err != nil {
		_, err := result.GetBool()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	currentBlock, err := parseHex(syncing["currentBlock"])
	if err != nil {
		return nil, err
	}

	highestBlock, err := parseHex(syncing["highestBlock"])
	if err != nil {
		return nil, err
	}

	knownStates, err := parseHex(syncing["knownStates"])
	if err != nil {
		return nil, err
	}

	pulledStates, err := parseHex(syncing["pulledStates"])
	if err != nil {
		return nil, err
	}

	startingBlock, err := parseHex(syncing["startingBlock"])
	if err != nil {
		return nil, err
	}

	return &Syncing{
		CurrentBlock:  currentBlock,
		HighestBlock:  highestBlock,
		KnownStates:   knownStates,
		PulledStates:  pulledStates,
		StartingBlock: startingBlock,
	}, nil
}

func (t *RpcClient) EthBlockNumber() (int64, error) {
	result, err := t.client.Call("eth_blockNumber")
	if err != nil {
		return 0, err
	}
	s, err := result.GetString()
	if err != nil {
		return 0, err
	}
	blockNumber, err := parseHex(s)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func explainNetVersion(version string) string {
	switch version {
	case "1":
		return "Mainnet"
	case "2":
		return "Testnet (Morden, deprecated!)"
	case "3":
		return "Testnet (Ropsten)"
	case "4":
		return "Testnet (Rinkeby)"
	case "42":
		return "Testnet (Kovan)"
	default:
		return version
	}
}

func (t *RpcClient) Close() error {
	return nil
}
