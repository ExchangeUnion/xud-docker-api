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

type RpcClient struct {
	btcConn   *grpc.ClientConn
	ltcConn   *grpc.ClientConn
	btcClient pb.BoltzClient
	ltcClient pb.BoltzClient
	btcMutex  *sync.Mutex
	ltcMutex  *sync.Mutex

	logger  *logrus.Entry
	service *core.SingleContainerService
}

func NewRpcClient(config config.RpcConfig, logger *logrus.Entry, service *core.SingleContainerService) *RpcClient {

	host := config["host"].(string)
	btcPort := uint16(config["btcPort"].(float64))
	ltcPort := uint16(config["ltcPort"].(float64))

	c := &RpcClient{
		btcConn:   nil,
		ltcConn:   nil,
		btcClient: nil,
		ltcClient: nil,
		btcMutex:  &sync.Mutex{},
		ltcMutex:  &sync.Mutex{},

		logger:  logger,
		service: service,
	}

	go c.lazyInit(host, btcPort, ltcPort)

	return c
}

func (t *RpcClient) createClient(client *pb.BoltzClient, _conn **grpc.ClientConn, mutex *sync.Mutex, host string, port uint16) {
	for {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		opts = append(opts, grpc.WithBlock())

		t.logger.Debug("Waiting for a running container")
		t.service.WaitContainerRunning()

		addr := fmt.Sprintf("%s:%d", host, port)
		t.logger.Debugf("Trying to connect with addr=%s", addr)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			cancel() // prevent context resource leak
			t.logger.Warnf("Failed to create gRPC connection: %s", err)
			time.Sleep(RpcRetryDelay)
			continue
		}
		cancel()

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
	_ = t.btcConn.Close()
	_ = t.ltcConn.Close()
	return nil
}

func (t *RpcClient) getRpcClient(currency string) pb.BoltzClient {
	currency = strings.ToLower(currency)
	switch currency {
	case "btc":
		t.btcMutex.Lock()
		defer t.btcMutex.Unlock()
		return t.btcClient
	case "ltc":
		t.ltcMutex.Lock()
		defer t.ltcMutex.Unlock()
		return t.ltcClient
	default:
		panic(errors.New("invalid currency: " + currency))
	}
}

func (t *RpcClient) GetServiceInfo(currency string) (*pb.GetServiceInfoResponse, error) {
	client := t.getRpcClient(currency)
	if client == nil {
		return nil, errors.New("no client")
	}
	fmt.Printf("client=%v", client)
	req := pb.GetServiceInfoRequest{}
	return client.GetServiceInfo(context.Background(), &req)
}

func (t *RpcClient) Deposit(currency string, inboundLiquidity uint32) (*pb.DepositResponse, error) {
	client := t.getRpcClient(currency)
	if client == nil {
		return nil, errors.New("no client")
	}
	req := pb.DepositRequest{}
	req.InboundLiquidity = inboundLiquidity
	return client.Deposit(context.Background(), &req)
}

func (t *RpcClient) Withdraw(currency string, amount int64, address string) (*pb.CreateReverseSwapResponse, error) {
	client := t.getRpcClient(currency)
	if client == nil {
		return nil, errors.New("no client")
	}
	req := pb.CreateReverseSwapRequest{}
	req.Amount = amount
	req.Address = address
	return client.CreateReverseSwap(context.Background(), &req)
}
