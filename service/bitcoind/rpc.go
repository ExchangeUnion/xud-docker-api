package bitcoind

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ybbus/jsonrpc"
	"net/http"
	"time"
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
		HTTPClient: &http.Client{
			Timeout: 3 * time.Second,
		},
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

func (t *RpcClient) GetBlockchainInfo(ctx context.Context) (*jsonrpc.RPCResponse, error) {
	// TODO jsonrpc call with context
	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled by context")
	default:
		response, err := t.client.Call("getblockchaininfo")
		if err != nil {
			return nil, err
		}
		return response, nil
	}
}
