package lnd

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	pb "github.com/ExchangeUnion/xud-docker-api/service/lnd/lnrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"sync"
	"time"
)

const (
	RpcRetryDelay = 3 * time.Second
)

type RpcClient struct {
	mutex   *sync.Mutex
	client  pb.LightningClient
	conn    *grpc.ClientConn
	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, logger *logrus.Entry, service *core.SingleContainerService) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)
	macaroon := config["macaroon"].(string)

	c := &RpcClient{
		mutex:   &sync.Mutex{},
		client:  nil,
		logger:  logger,
		service: service,
	}

	go c.lazyInit(host, port, tlsCert, macaroon)

	return c
}

func (t *RpcClient) lazyInit(host string, port uint16, tlsCert string, macaroon string) {
	for {
		creds, err := credentials.NewClientTLSFromFile(tlsCert, "localhost")
		if err != nil {
			t.logger.Warnf("Failed to create gRPC TLS credentials: %s", err)
			time.Sleep(RpcRetryDelay)
			continue
		}

		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())

		if _, err := os.Stat(macaroon); os.IsNotExist(err) {
			t.logger.Warnf("Waiting for %s", macaroon)
			time.Sleep(RpcRetryDelay)
			continue
		}

		opts = append(opts, grpc.WithPerRPCCredentials(&MacaroonCredential{Readonly: macaroon}))

		t.logger.Debug("Waiting for a running container")
		t.service.WaitContainerRunning()

		addr := fmt.Sprintf("%s:%d", host, port)
		t.logger.Debugf("Trying to connect with addr=%s tlsCert=%s macaroon=%s", addr, tlsCert, macaroon)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			cancel() // prevent context resource leak
			t.logger.Warnf("Failed to create gRPC connection: %s", err)
			time.Sleep(RpcRetryDelay)
			continue
		}
		cancel() // prevent context resource leak

		t.logger.Debugf("Created gRPC connection")
		t.conn = conn

		t.mutex.Lock()
		t.client = pb.NewLightningClient(conn)
		t.mutex.Unlock()

		break
	}
}

func (t *RpcClient) Close() error {
	_ = t.conn.Close()
	return nil
}

func (t *RpcClient) GetInfo() (*pb.GetInfoResponse, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.client == nil {
		return nil, errors.New("no client")
	}

	req := pb.GetInfoRequest{}
	return t.client.GetInfo(context.Background(), &req)
}
