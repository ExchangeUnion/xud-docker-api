package lnd

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/rpc"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	pb "github.com/ExchangeUnion/xud-docker-api/service/lnd/lnrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	errNoClient     = errors.New("no client")
)

type RpcClient struct {
	conn *rpc.GrpcConn

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)
	macaroon := config["macaroon"].(string)

	logger := service.GetLogger().WithField("name", fmt.Sprintf("service.%s.rpc", service.GetName()))

	conn := rpc.NewGrpcConn(host, port, tlsCert, macaroon, logger, func(conn *grpc.ClientConn) interface{}{
		return pb.NewLightningClient(conn)
	})

	go conn.Open()

	return &RpcClient{
		conn: conn,
		logger: logger,
		service: service,
	}
}

func (t *RpcClient) Close() error {
	if err := t.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (t *RpcClient) getClient() (pb.LightningClient, error) {
	client := t.conn.GetClient()
	if client == nil {
		return nil, errNoClient
	}
	return client.(pb.LightningClient), nil
}

func (t *RpcClient) GetInfo(ctx context.Context) (*pb.GetInfoResponse, error) {
	client, err := t.getClient()
	if err != nil {
		return nil, err
	}
	req := pb.GetInfoRequest{}
	return client.GetInfo(ctx, &req)
}
