package boltz

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/boltz/boltzrpc"
	"google.golang.org/grpc"
	"strings"
	"sync"
	"time"
)

type RpcClient struct {
	cond *sync.Cond
	btcClient pb.BoltzClient
	ltcClient pb.BoltzClient
	btcConn *grpc.ClientConn
	ltcConn *grpc.ClientConn
}

func NewRpcClient(config config.RpcConfig) *RpcClient {

	host := config["host"].(string)
	btcPort := uint16(config["btcPort"].(float64))
	ltcPort := uint16(config["ltcPort"].(float64))

	c := &RpcClient{
		cond:   sync.NewCond(&sync.Mutex{}),
		btcClient: nil,
		ltcClient: nil,
	}

	go c.lazyInit(host, btcPort, ltcPort)

	return c
}

func (t *RpcClient) createClient(client *pb.BoltzClient, _conn **grpc.ClientConn, host string, port uint16) {
	for {
		addr := fmt.Sprintf("%s:%d", host, port)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithBlock())

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		*_conn = conn

		t.cond.L.Lock()
		*client = pb.NewBoltzClient(conn)
		t.cond.Broadcast()
		t.cond.L.Unlock()

		break
	}
}

func (t *RpcClient) lazyInit(host string, btcPort uint16, ltcPort uint16) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		t.createClient(&t.btcClient, &t.btcConn, host, btcPort)
		wg.Done()
	}()

	go func() {
		t.createClient(&t.ltcClient, &t.ltcConn, host, ltcPort)
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
		t.cond.L.Lock()
		for t.btcClient == nil {
			t.cond.Wait()
		}
		defer t.cond.L.Unlock()
		return t.btcClient
	case "ltc":
		t.cond.L.Lock()
		for t.btcClient == nil {
			t.cond.Wait()
		}
		defer t.cond.L.Unlock()
		return t.ltcClient
	default:
		panic(errors.New("invalid currency: " + currency))
	}
}

func (t *RpcClient) GetServiceInfo(currency string) (*pb.GetServiceInfoResponse, error) {
	client := t.getRpcClient(currency)
	req := pb.GetServiceInfoRequest{}
	return client.GetServiceInfo(context.Background(), &req)
}

func (t *RpcClient) Deposit(currency string, inboundLiquidity uint32) (*pb.DepositResponse, error) {
	client := t.getRpcClient(currency)
	req := pb.DepositRequest{}
	req.InboundLiquidity = inboundLiquidity
	return client.Deposit(context.Background(), &req)
}

func (t *RpcClient) Withdraw(currency string, amount int64, address string) (*pb.CreateReverseSwapResponse, error) {
	client := t.getRpcClient(currency)
	req := pb.CreateReverseSwapRequest{}
	req.Amount = amount
	req.Address = address
	return client.CreateReverseSwap(context.Background(), &req)
}