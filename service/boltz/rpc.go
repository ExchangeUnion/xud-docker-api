package boltz

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/rpc"
	pb "github.com/ExchangeUnion/xud-docker-api/service/boltz/boltzrpc"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
)

var (
	errNoClient = errors.New("no client")
)

type RpcClient struct {
	btcConn *rpc.GrpcConn
	ltcConn *rpc.GrpcConn

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func newGrpcConn(config map[string]interface{}, logger *logrus.Entry) *rpc.GrpcConn {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)
	macaroon := config["macaroon"].(string)

	conn := rpc.NewGrpcConn(host, port, tlsCert, macaroon, logger, func(conn *grpc.ClientConn) interface{} {
		return pb.NewBoltzClient(conn)
	})
	return conn
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {
	bitcoin := config["bitcoin"].(map[string]interface{})
	litecoin := config["litecoin"].(map[string]interface{})

	logger := service.GetLogger().WithField("name", fmt.Sprintf("service.%s.rpc", service.GetName()))

	btcConn := newGrpcConn(bitcoin, logger)
	ltcConn := newGrpcConn(litecoin, logger)

	go btcConn.Open()
	go ltcConn.Open()

	c := &RpcClient{
		btcConn: btcConn,
		ltcConn: ltcConn,
		logger:  logger,
		service: service,
	}

	return c
}

func (t *RpcClient) Close() error {
	if err := t.btcConn.Close(); err != nil {
		return err
	}
	if err := t.ltcConn.Close(); err != nil {
		return err
	}
	return nil
}

func (t *RpcClient) getRpcClient(currency string) (pb.BoltzClient, error) {
	currency = strings.ToLower(currency)
	var client pb.BoltzClient
	switch currency {
	case "btc":
		client = t.btcConn.GetClient().(pb.BoltzClient)
	case "ltc":
		client = t.ltcConn.GetClient().(pb.BoltzClient)
	default:
		panic(errors.New("invalid currency: " + currency))
	}
	if client == nil {
		return nil, errNoClient
	}
	return client, nil
}

func (t *RpcClient) GetServiceInfo(ctx context.Context, currency string) (*pb.GetServiceInfoResponse, error) {
	client, err := t.getRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.GetServiceInfoRequest{}
	return client.GetServiceInfo(ctx, &req)
}

func (t *RpcClient) Deposit(ctx context.Context, currency string, inboundLiquidity uint32) (*pb.DepositResponse, error) {
	client, err := t.getRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.DepositRequest{}
	req.InboundLiquidity = inboundLiquidity
	return client.Deposit(ctx, &req)
}

func (t *RpcClient) Withdraw(ctx context.Context, currency string, amount int64, address string) (*pb.CreateReverseSwapResponse, error) {
	client, err := t.getRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.CreateReverseSwapRequest{}
	req.Amount = amount
	req.Address = address
	return client.CreateReverseSwap(ctx, &req)
}
