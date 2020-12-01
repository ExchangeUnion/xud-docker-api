package xud

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud/xudrpc"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/xud/xudrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
	"time"
)

type RpcClient struct {
	cond   *sync.Cond
	client xudrpc.XudClient
	conn  *grpc.ClientConn
}

func NewRpcClient(config config.RpcConfig) *RpcClient {

	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)

	c := &RpcClient{
		cond:   sync.NewCond(&sync.Mutex{}),
		client: nil,
	}

	go c.lazyInit(host, port, tlsCert)

	return c
}

func (t *RpcClient) lazyInit(host string, port uint16, tlsCert string) {
	for {
		creds, err := credentials.NewClientTLSFromFile(tlsCert, "localhost")
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		addr := fmt.Sprintf("%s:%d", host, port)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		t.conn = conn

		t.cond.L.Lock()
		t.client = pb.NewXudClient(conn)
		t.cond.Broadcast()
		t.cond.L.Unlock()

		break
	}
}

func (t *RpcClient) GetInfo() (*pb.GetInfoResponse, error) {
	t.cond.L.Lock()
	for t.client == nil {
		t.cond.Wait()
	}
	defer t.cond.L.Unlock()

	req := pb.GetInfoRequest{}
	return t.client.GetInfo(context.Background(), &req)
}

func (t *RpcClient) GetBalance(currency string) (*pb.GetBalanceResponse, error) {
	t.cond.L.Lock()
	for t.client == nil {
		t.cond.Wait()
	}
	defer t.cond.L.Unlock()

	req := pb.GetBalanceRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return t.client.GetBalance(context.Background(), &req)
}

func (t *RpcClient) GetTradeHistory(limit uint32) (*pb.TradeHistoryResponse, error) {
	t.cond.L.Lock()
	for t.client == nil {
		t.cond.Wait()
	}
	defer t.cond.L.Unlock()

	req := pb.TradeHistoryRequest{}
	if limit != 0 {
		req.Limit = limit
	}
	return t.client.TradeHistory(context.Background(), &req)
}

func (t *RpcClient) GetTradingLimits(currency string) (*pb.TradingLimitsResponse, error) {
	t.cond.L.Lock()
	for t.client == nil {
		t.cond.Wait()
	}
	defer t.cond.L.Unlock()

	req := pb.TradingLimitsRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return t.client.TradingLimits(context.Background(), &req)
}

func (t *RpcClient) Close() error {
	_ = t.conn.Close()
	return nil
}


