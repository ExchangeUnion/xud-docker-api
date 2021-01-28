package xud

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/rpc"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/ExchangeUnion/xud-docker-api/service/xud/xudrpc"
	pb "github.com/ExchangeUnion/xud-docker-api/service/xud/xudrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	errNoClient     = errors.New("no client")
	errNoInitClient = errors.New("no init client")
)

type RpcClient struct {
	conn *rpc.GrpcConn

	logger  *logrus.Entry
	service *core.SingleContainerService
}

type Clients struct {
	Client     pb.XudClient
	InitClient pb.XudInitClient
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)

	logger := service.GetLogger().WithField("name", fmt.Sprintf("service.%s.rpc", service.GetName()))

	conn := rpc.NewGrpcConn(host, port, tlsCert, "", logger, func(conn *grpc.ClientConn) interface{} {
		return Clients{
			Client:     pb.NewXudClient(conn),
			InitClient: pb.NewXudInitClient(conn),
		}
	})

	go conn.Open()

	return &RpcClient{
		conn:    conn,
		logger:  logger,
		service: service,
	}
}

func (t *RpcClient) Close() error {
	if err := t.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (t *RpcClient) getClient() (xudrpc.XudClient, error) {
	clients := t.conn.GetClient()
	if clients == nil {
		return nil, errNoClient
	}
	return clients.(Clients).Client, nil
}

func (t *RpcClient) getInitClient() (xudrpc.XudInitClient, error) {
	clients := t.conn.GetClient()
	if clients == nil {
		return nil, errNoInitClient
	}
	return clients.(Clients).InitClient, nil
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

func (t *RpcClient) OrderBook(ctx context.Context, pairId string, precision int32, limit uint32) (*pb.OrderBookResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.OrderBookRequest{
		PairId:    pairId,
		Precision: precision,
		Limit:     limit,
	}
	return client.OrderBook(ctx, &req)
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
