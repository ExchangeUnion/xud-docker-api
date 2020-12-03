package lnd

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/lnd/lnrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
	"time"
)

type RpcClient struct {
	mutex   *sync.Mutex
	client pb.LightningClient
	conn *grpc.ClientConn
	logger *logrus.Entry
}

func NewRpcClient(config config.RpcConfig, logger *logrus.Entry) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)
	macaroon := config["macaroon"].(string)

	c := &RpcClient{
		mutex:   &sync.Mutex{},
		client: nil,
		logger: logger,
	}

	go c.lazyInit(host, port, tlsCert, macaroon)

	return c
}

func (t *RpcClient) lazyInit(host string, port uint16, tlsCert string, macaroon string) {
	for {
		creds, err := credentials.NewClientTLSFromFile(tlsCert, "localhost")
		if err != nil {
			t.logger.Warnf("Failed to create gRPC TLS credentials: %s (will retry in 3 seconds)", err)
			time.Sleep(3 * time.Second)
			continue
		}

		addr := fmt.Sprintf("%s:%d", host, port)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())

		opts = append(opts, grpc.WithPerRPCCredentials(&MacaroonCredential{Readonly: macaroon}))

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			t.logger.Warnf("Failed to create gRPC connection: %s (will retry in 3 seconds)", err)
			time.Sleep(3 * time.Second)
			continue
		}

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
