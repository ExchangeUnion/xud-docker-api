package xud

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/xud/xudrpc"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type XudService struct {
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
	rpcClient  pb.XudClient
	conn       *grpc.ClientConn
}

type XudRpc struct {
	Host string
	Port int
	Cert string
}

func New(name string, containerName string) *XudService {

	return &XudService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *XudService) GetInfo() (*pb.GetInfoResponse, error) {
	client, err := t.getRpcClient()
	if err != nil {
		return nil, err
	}

	req := pb.GetInfoRequest{}
	return client.GetInfo(context.Background(), &req)
}

func (t *XudService) GetBalance(currency string) (*pb.GetBalanceResponse, error) {
	client, err := t.getRpcClient()
	if err != nil {
		return nil, err
	}

	req := pb.GetBalanceRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return client.GetBalance(context.Background(), &req)
}

func (t *XudService) GetTradeHistory(limit uint32) (*pb.TradeHistoryResponse, error) {
	client, err := t.getRpcClient()
	if err != nil {
		return nil, err
	}

	req := pb.TradeHistoryRequest{}
	if limit != 0 {
		req.Limit = limit
	}
	return client.TradeHistory(context.Background(), &req)
}

func (t *XudService) GetTradingLimits(currency string) (*pb.TradingLimitsResponse, error) {
	client, err := t.getRpcClient()
	if err != nil {
		return nil, err
	}

	req := pb.TradingLimitsRequest{}
	if currency != "" {
		req.Currency = currency
	}
	return client.TradingLimits(context.Background(), &req)
}

func (t *XudService) ConfigureRpc(options *service.RpcOptions) {
	t.rpcOptions = options
}

func (t *XudService) getRpcClient() (pb.XudClient, error) {
	if t.rpcClient == nil {
		creds, err := credentials.NewClientTLSFromFile(t.rpcOptions.TlsCert, "localhost")
		if err != nil {
			return nil, err
		}

		addr := fmt.Sprintf("%s:%d", t.rpcOptions.Host, t.rpcOptions.Port)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())
		//opts = append(opts, grpc.WithTimeout(time.Duration(10000)))

		conn, err := grpc.Dial(addr, opts...)
		if err != nil {
			return nil, err
		}

		t.rpcClient = pb.NewXudClient(conn)
	}
	return t.rpcClient, nil
}

func (t *XudService) ConfigureRouter(r *gin.Engine) {
	r.GET("/api/v1/xud/getinfo", func(c *gin.Context) {
		resp, err := t.GetInfo()
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
	})
	r.GET("/api/v1/xud/getbalance", func(c *gin.Context) {
		resp, err := t.GetBalance("")
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
	})
	r.GET("/api/v1/xud/getbalance/:currency", func(c *gin.Context) {
		resp, err := t.GetBalance(c.Param("currency"))
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
	})
	r.GET("/api/v1/xud/tradehistory", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "0")
		limit, err := strconv.ParseUint(limitStr, 10, 32)
		if err != nil {
			msg := fmt.Sprintf("invalid limit: %s", err.Error())
			utils.JsonError(c, msg, http.StatusBadRequest)
			return
		}
		if limit < 0 {
			msg := fmt.Sprintf("invalid limit: %d", limit)
			utils.JsonError(c, msg, http.StatusBadRequest)
			return
		}
		resp, err := t.GetTradeHistory(uint32(limit))
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
	})
	r.GET("/api/v1/xud/tradinglimits", func(c *gin.Context) {
		resp, err := t.GetTradingLimits("")
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
	})
	r.GET("/api/v1/xud/tradinglimits/:currency", func(c *gin.Context) {
		resp, err := t.GetTradingLimits(c.Param("currency"))
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
	})
}

func (t *XudService) Close() {
	err := t.conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (t *XudService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		resp, err := t.GetInfo()
		if err != nil {
			if strings.Contains(err.Error(), "xud is locked") {
				return "Wallet locked. Unlock with xucli unlock.", nil
			} else if strings.Contains(err.Error(), "no such file or directory, open '/root/.xud/tls.cert'") {
				return "Starting...", nil
			} else if strings.Contains(err.Error(), "xud is starting") {
				return "Starting...", nil
			}
			return "", err
		}
		lndbtcStatus := resp.Lnd["BTC"].Status
		lndltcStatus := resp.Lnd["LTC"].Status
		connextStatus := resp.Connext.Status

		if lndbtcStatus == "Ready" && lndltcStatus == "Ready" && connextStatus == "Ready" {
			return "Ready", nil
		}

		if strings.Contains(lndbtcStatus, "has no active channels") ||
			strings.Contains(lndltcStatus, "has no active channels") ||
			strings.Contains(connextStatus, "has no active channels") {
			return "Waiting for channels", nil
		}

		var notReady []string
		if lndbtcStatus != "Ready" {
			notReady = append(notReady, "lndbtc")
		}
		if lndltcStatus != "Ready" {
			notReady = append(notReady, "lndltc")
		}
		if connextStatus != "Ready" {
			notReady = append(notReady, "connext")
		}

		return "Waiting for " + strings.Join(notReady, ", "), nil
	} else {
		return status, nil
	}
}
