package lnd

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
	"time"
)

type RpcClient struct {
	cond   *sync.Cond
	client pb.LightningClient
	conn *grpc.ClientConn
}

func NewRpcClient(config config.RpcConfig) *RpcClient {
	host := config["host"].(string)
	port := uint16(config["port"].(float64))
	tlsCert := config["tlsCert"].(string)
	macaroon := config["macaroon"].(string)

	c := &RpcClient{
		cond:   sync.NewCond(&sync.Mutex{}),
		client: nil,
	}

	go c.lazyInit(host, port, tlsCert, macaroon)

	return c
}

func (t *RpcClient) lazyInit(host string, port uint16, tlsCert string, macaroon string) {
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

		opts = append(opts, grpc.WithPerRPCCredentials(&MacaroonCredential{Readonly: macaroon}))

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		t.conn = conn

		t.cond.L.Lock()
		t.client = pb.NewLightningClient(conn)
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

func (t *RpcClient) Close() error {
	_ = t.conn.Close()
	return nil
}
