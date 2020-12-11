package boltz

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	pb "github.com/ExchangeUnion/xud-docker-api/service/boltz/boltzrpc"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
	"sync"
	"time"
)

const (
	RpcRetryDelay = 3 * time.Second
)

var (
	NoClient = errors.New("no client")
)

type RpcClient struct {
	btcConn   *grpc.ClientConn
	ltcConn   *grpc.ClientConn
	btcClient pb.BoltzClient
	ltcClient pb.BoltzClient
	btcMutex  *sync.RWMutex
	ltcMutex  *sync.RWMutex

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, service *core.SingleContainerService) *RpcClient {

	host := config["host"].(string)
	btcPort := uint16(config["btcPort"].(float64))
	ltcPort := uint16(config["ltcPort"].(float64))

	c := &RpcClient{
		btcConn:   nil,
		ltcConn:   nil,
		btcClient: nil,
		ltcClient: nil,
		btcMutex:  &sync.RWMutex{},
		ltcMutex:  &sync.RWMutex{},

		logger:  service.GetLogger().WithField("scope", "RPC"),
		service: service,
	}

	go c.lazyInit(host, btcPort, ltcPort)

	return c
}

func (t *RpcClient) createClient(client *pb.BoltzClient, _conn **grpc.ClientConn, mutex *sync.RWMutex, host string, port uint16) {
	for {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		opts = append(opts, grpc.WithBlock())

		t.logger.Debug("Waiting for a running container")
		t.service.WaitContainerRunning()

		addr := fmt.Sprintf("%s:%d", host, port)
		t.logger.Debugf("Trying to connect with addr=%s", addr)

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		//cancel() // prevent context resource leak
		if err != nil {
			t.logger.Warnf("Failed to create gRPC connection: %s", err)
			time.Sleep(RpcRetryDelay)
			continue
		}

		t.logger.Debugf("Created gRPC connection")

		*_conn = conn

		mutex.Lock()
		*client = pb.NewBoltzClient(conn)
		mutex.Unlock()

		break
	}
}

func (t *RpcClient) lazyInit(host string, btcPort uint16, ltcPort uint16) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		t.createClient(&t.btcClient, &t.btcConn, t.btcMutex, host, btcPort)
		wg.Done()
	}()

	go func() {
		t.createClient(&t.ltcClient, &t.ltcConn, t.ltcMutex, host, ltcPort)
		wg.Done()
	}()

	wg.Wait()
}

func (t *RpcClient) Close() error {
	var err error

	err = t.btcConn.Close()
	if err != nil {
		return err
	}

	err = t.ltcConn.Close()
	if err != nil {
		return err
	}

	return nil
}

func (t *RpcClient) getRpcClient(currency string) (pb.BoltzClient, error) {
	currency = strings.ToLower(currency)
	var client pb.BoltzClient
	switch currency {
	case "btc":
		t.btcMutex.RLock()
		defer t.btcMutex.RUnlock()
		client = t.btcClient
	case "ltc":
		t.ltcMutex.RLock()
		defer t.ltcMutex.RUnlock()
		client = t.ltcClient
	default:
		panic(errors.New("invalid currency: " + currency))
	}
	if client == nil {
		return nil, NoClient
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
