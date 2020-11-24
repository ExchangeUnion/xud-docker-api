package boltz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/boltz/boltzrpc"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RpcOptions struct {
	Host    string
	BtcPort int16
	LtcPort int16
}

type BoltzService struct {
	*service.SingleContainerService
	rpcOptions   *RpcOptions
	btcRpcClient pb.BoltzClient
	ltcRpcClient pb.BoltzClient
	conn         *grpc.ClientConn
}

type Node string

const (
	BTC Node = "btc"
	LTC Node = "ltc"
)

func New(
	name string,
	containerName string,
) *BoltzService {
	return &BoltzService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

// {
//  "symbol": "BTC",
//  "lnd_pubkey": "02c882fbd75ba7c0e3175a0b86037b4d056599a694fcfad56589fc05d081b62774",
//  "block_height": 1835961
// }

func (t *BoltzService) GetInfo(node Node) (map[string]interface{}, error) {
	output, err := t.Exec1([]string{"wrapper", string(node), "getinfo"})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type NodeStatus struct {
	Status string
	IsUp   bool
}

func (t *BoltzService) checkNode(node Node) NodeStatus {
	_, err := t.GetInfo(node)
	if err == nil {
		return NodeStatus{Status: string(node) + " up", IsUp: true}
	} else {
		return NodeStatus{Status: string(node) + " down", IsUp: false}
	}
}

func (t *BoltzService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status != "Container running" {
		return status, err
	}

	btcStatus := t.checkNode(BTC)
	ltcStatus := t.checkNode(LTC)

	if btcStatus.IsUp && ltcStatus.IsUp {
		return "Ready", nil
	} else {
		return btcStatus.Status + "; " + ltcStatus.Status, nil
	}
}

func (t *BoltzService) GetServiceInfo(currency string) (*pb.GetServiceInfoResponse, error) {
	client, err := t.GetRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.GetServiceInfoRequest{}
	return client.GetServiceInfo(context.Background(), &req)
}

func (t *BoltzService) Deposit(currency string, inboundLiquidity uint32) (*pb.DepositResponse, error) {
	client, err := t.GetRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.DepositRequest{}
	req.InboundLiquidity = inboundLiquidity
	return client.Deposit(context.Background(), &req)
}

func (t *BoltzService) Withdraw(currency string, amount int64, address string) (*pb.CreateReverseSwapResponse, error) {
	client, err := t.GetRpcClient(currency)
	if err != nil {
		return nil, err
	}
	req := pb.CreateReverseSwapRequest{}
	req.Amount = amount
	req.Address = address
	return client.CreateReverseSwap(context.Background(), &req)
}

func (t *BoltzService) ConfigureRpc(options *RpcOptions) {
	t.rpcOptions = options
}

func (t *BoltzService) GetRpcClient(currency string) (pb.BoltzClient, error) {
	var client pb.BoltzClient
	var err error
	if strings.ToLower(currency) == "btc" {
		client, err = t.GetBtcRpcClient()
	} else if strings.ToLower(currency) == "ltc" {
		client, err = t.GetLtcRpcClient()
	} else {
		err = errors.New(fmt.Sprintf("Currency %s is not supported", currency))
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (t *BoltzService) CreateRpcClient(port int16) (pb.BoltzClient, error) {
	addr := fmt.Sprintf("%s:%d", t.rpcOptions.Host, port)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			return nil, errors.New("cannot establish gRPC connection")
		}
		return nil, err
	}

	return pb.NewBoltzClient(conn), nil
}

func (t *BoltzService) GetBtcRpcClient() (pb.BoltzClient, error) {
	if t.btcRpcClient == nil {
		client, err := t.CreateRpcClient(t.rpcOptions.BtcPort)
		if err != nil {
			return nil, err
		}
		t.btcRpcClient = client
	}
	return t.btcRpcClient, nil
}

func (t *BoltzService) GetLtcRpcClient() (pb.BoltzClient, error) {
	if t.ltcRpcClient == nil {
		client, err := t.CreateRpcClient(t.rpcOptions.LtcPort)
		if err != nil {
			return nil, err
		}
		t.ltcRpcClient = client
	}
	return t.ltcRpcClient, nil
}

func (t *BoltzService) ConfigureRouter(r *gin.RouterGroup) {
	r.GET("/v1/boltz/service-info/:currency", func(c *gin.Context) {
		resp, err := t.GetServiceInfo(c.Param("currency"))
		t.HandleProtobufResponse(c, resp, err)
	})
	r.GET("/v1/boltz/deposit/:currency", func(c *gin.Context) {
		inboundLiquidity, err := strconv.Atoi(c.DefaultQuery("inbound_liquidity", "50"))
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid value %s for inbound_liquidity", c.Query("inbound_liquidity")), http.StatusBadRequest)
			return
		}
		resp, err := t.Deposit(c.Param("currency"), uint32(inboundLiquidity))
		t.HandleProtobufResponse(c, resp, err)
	})
	r.POST("/v1/boltz/withdraw/:currency", func(c *gin.Context) {
		amount, err := strconv.ParseInt(c.PostForm("amount"), 10, 64)
		if err != nil {
			utils.JsonError(c, fmt.Sprintf("Invalid amount %s", c.PostForm("amount")), http.StatusBadRequest)
			return
		}
		resp, err := t.Withdraw(c.Param("currency"), amount, c.PostForm("address"))
		t.HandleProtobufResponse(c, resp, err)
	})
}

func (t *BoltzService) HandleProtobufResponse(c *gin.Context, resp proto.Message, err error) {
	if err != nil {
		utils.JsonError(c, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{EmitDefaults: true}
	err = m.Marshal(c.Writer, resp)
	if err != nil {
		utils.JsonError(c, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
}

func (t *BoltzService) Close() {
	err := t.conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}
