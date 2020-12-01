package bitcoind

import (
	"encoding/base64"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ybbus/jsonrpc"
)

type Fork struct {
	Type   string
	Active bool
	Height int32
}

type BlockchainInfo struct {
	Chain                string
	Blocks               int32
	Headers              int32
	BestBlockHash        string
	Difficulty           float64
	MedianTime           int32
	VerificationProgress float32
	InitialBlockDownload bool
	ChainWork            string
	SizeOnDisk           int32
	Pruned               bool
	SoftForks            map[string]Fork
	Warnings             string
}

type RpcClient struct {
	client jsonrpc.RPCClient
}

func NewRpcClient(config config.RpcConfig) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))

	addr := fmt.Sprintf("http://%s:%d", host, port)
	client := jsonrpc.NewClientWithOpts(addr, &jsonrpc.RPCClientOpts{
		CustomHeaders: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("xu"+":"+"xu")),
		},
	})

	return &RpcClient{
		client: client,
	}
}

func (t *RpcClient) Close() error {
	return nil
}

func (t *RpcClient) GetBlockchainInfo() (*jsonrpc.RPCResponse, error) {
	response, err := t.client.Call("getblockchaininfo")
	if err != nil {
		return nil, err
	}
	return response, nil
}
