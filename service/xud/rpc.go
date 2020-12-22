package xud

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/ExchangeUnion/xud-docker-api/service/xud/xudrpc"
	pb "github.com/ExchangeUnion/xud-docker-api/service/xud/xudrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
	"time"
)

const (
	RetryDelay         = 3 * time.Second
	GrpcConnectTimeout = 3 * time.Second
)

var (
	NoClient     = errors.New("no client")
	NoInitClient = errors.New("no init client")
)

type RpcClient struct {
	client     xudrpc.XudClient
	initClient xudrpc.XudInitClient
	conn       *grpc.ClientConn
	mutex      *sync.RWMutex

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {

	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)

	c := &RpcClient{
		client:     nil,
		initClient: nil,
		conn:       nil,
		mutex:      &sync.RWMutex{},

		logger:  service.GetLogger().WithField("scope", "RPC"),
		service: service,
	}

	go c.lazyInit(host, port, tlsCert)

	return c
}

func (t *RpcClient) lazyInit(host string, port uint16, tlsCert string) {
	for {
		creds, err := credentials.NewClientTLSFromFile(tlsCert, "localhost")
		if err != nil {
			time.Sleep(RetryDelay)
			continue
		}

		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())

		t.logger.Debug("Waiting for a running container")
		t.service.WaitContainerRunning()

		ctx, cancel := context.WithTimeout(context.Background(), GrpcConnectTimeout)
		addr := fmt.Sprintf("%s:%d", host, port)
		t.logger.Debugf("Trying to connect with addr=%s tlsCert=%s macaroon=%s", addr, tlsCert)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		cancel() // TODO make sure this won't close the conn
		if err != nil {
			t.logger.Warnf("Failed to create gRPC connection: %s", err)
			time.Sleep(RetryDelay)
			continue
		}

		t.logger.Debugf("Created gRPC connection")
		t.conn = conn

		t.mutex.Lock()
		t.client = pb.NewXudClient(conn)
		t.initClient = pb.NewXudInitClient(conn)
		t.mutex.Unlock()

		break
	}
}

func (t *RpcClient) Close() error {
	err := t.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (t *RpcClient) getClient() (xudrpc.XudClient, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.client == nil {
		return nil, NoClient
	}
	return t.client, nil
}

func (t *RpcClient) getInitClient() (xudrpc.XudInitClient, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.client == nil {
		return nil, NoInitClient
	}
	return t.initClient, nil
}

func (t *RpcClient) GetInfo(ctx context.Context) (*pb.GetInfoResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.GetInfoRequest{}
	return client.GetInfo(ctx, &req)
}

func (t *RpcClient) GetBalance(ctx context.Context, currency string) (*pb.GetBalanceResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.GetBalanceRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return client.GetBalance(ctx, &req)
}

func (t *RpcClient) GetTradeHistory(ctx context.Context, limit uint32) (*pb.TradeHistoryResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.TradeHistoryRequest{}
	if limit != 0 {
		req.Limit = limit
	}
	return client.TradeHistory(ctx, &req)
}

func (t *RpcClient) GetTradingLimits(ctx context.Context, currency string) (*pb.TradingLimitsResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.TradingLimitsRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return client.TradingLimits(ctx, &req)
}

func (t *RpcClient) CreateNode(ctx context.Context, password string) (*pb.CreateNodeResponse, error) {
	client, err := t.getInitClient()
	if err != nil {
		return nil, err
	}
	req := pb.CreateNodeRequest{Password: password}
	return client.CreateNode(ctx, &req)
}

func (t *RpcClient) RestoreNode(ctx context.Context, password string, seedMnemonic []string, lndBackups map[string][]byte, xudDatabase []byte) (*pb.RestoreNodeResponse, error) {
	client, err := t.getInitClient()
	if err != nil {
		return nil, err
	}
	req := pb.RestoreNodeRequest{
		Password:     password,
		SeedMnemonic: seedMnemonic,
		LndBackups:   lndBackups,
		XudDatabase:  xudDatabase,
	}
	return client.RestoreNode(ctx, &req)
}

func (t *RpcClient) UnlockNode(ctx context.Context, password string) (*pb.UnlockNodeResponse, error) {
	client, err := t.getInitClient()
	if err != nil {
		return nil, err
	}
	req := pb.UnlockNodeRequest{
		Password: password,
	}
	return client.UnlockNode(ctx, &req)
}

func (t *RpcClient) ChangePassword(ctx context.Context, newPassword string, oldPassword string) (*pb.ChangePasswordResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.ChangePasswordRequest{
		NewPassword: newPassword,
		OldPassword: oldPassword,
	}
	return client.ChangePassword(ctx, &req)
}

func (t *RpcClient) GetMnemonic(ctx context.Context) (*pb.GetMnemonicResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.GetMnemonicRequest{}
	return client.GetMnemonic(ctx, &req)
}

func (t *RpcClient) ListPairs(ctx context.Context) (*pb.ListPairsResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.ListPairsRequest{}
	return client.ListPairs(ctx, &req)
}

func (t *RpcClient) ListOrders(ctx context.Context, pairId string, owner pb.ListOrdersRequest_Owner, limit uint32, includeAliases bool) (*pb.ListOrdersResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.ListOrdersRequest{
		PairId:         pairId,
		Owner:          owner,
		Limit:          limit,
		IncludeAliases: includeAliases,
	}
	return client.ListOrders(ctx, &req)
}

func (t *RpcClient) PlaceOrder(ctx context.Context, pairId string, side pb.OrderSide, price float64, quantity uint64, orderId string, replaceOrderId string, immediateOrCancel bool) (*pb.PlaceOrderResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.PlaceOrderRequest{
		PairId:            pairId,
		Side:              side,
		Price:             price,
		Quantity:          quantity,
		OrderId:           orderId,
		ReplaceOrderId:    replaceOrderId,
		ImmediateOrCancel: immediateOrCancel,
	}
	return client.PlaceOrderSync(ctx, &req)
}

func (t *RpcClient) RemoveOrder(ctx context.Context, orderId string, quantity uint64) (*pb.RemoveOrderResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.RemoveOrderRequest{
		OrderId:  orderId,
		Quantity: quantity,
	}
	return client.RemoveOrder(ctx, &req)
}
