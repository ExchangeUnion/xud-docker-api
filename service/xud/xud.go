package xud

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/xud/xudrpc"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
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

func (t *XudService) GetBalance(w http.ResponseWriter, r *http.Request) {
	client, err := t.getRpcClient()
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req := pb.GetBalanceRequest{}
	if currency, ok := mux.Vars(r)["currency"]; ok {
		req.Currency = currency
	}
	resp, err := client.GetBalance(context.Background(), &req)
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{EmitDefaults: true}
	err = m.Marshal(w, resp)
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func (t *XudService) GetTradeHistory(w http.ResponseWriter, r *http.Request) {
	client, err := t.getRpcClient()
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req := pb.TradeHistoryRequest{}
	if limit, ok := mux.Vars(r)["limit"]; ok {
		i, err := strconv.ParseUint(limit, 10, 32)
		if err != nil {
			msg := fmt.Sprintf("invalid limit: %s", err.Error())
			utils.JsonError(w, msg, http.StatusBadRequest)
			return
		}
		req.Limit = uint32(i)
	}
	resp, err := client.TradeHistory(context.Background(), &req)
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{EmitDefaults: true}
	err = m.Marshal(w, resp)
	if err != nil {
		utils.JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func (t *XudService) ConfigureRpc(options *service.RpcOptions) {
	t.rpcOptions = options
}

func (t *XudService) getRpcClient() (pb.XudClient, error) {
	if t.rpcClient == nil {
		tlsFile, ok := t.rpcOptions.Credential.(service.TlsFileCredential)
		if !ok {
			return nil, errors.New("TlsFileCredential is required")
		}

		creds, err := credentials.NewClientTLSFromFile(tlsFile.File, "localhost")
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

func (t *XudService) ConfigureRouter(r *mux.Router) {
	t.SingleContainerService.ConfigureRouter(r)
	r.HandleFunc("/api/v1/xud/getinfo", func(w http.ResponseWriter, r *http.Request) {
		resp, err := t.GetInfo()
		if err != nil {
			utils.JsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(w, resp)
		if err != nil {
			utils.JsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}).Methods("GET")
	r.HandleFunc("/api/v1/xud/getbalance", t.GetBalance).Methods("GET")
	r.HandleFunc("/api/v1/xud/getbalance/{currency}", t.GetBalance).Methods("GET")
	r.HandleFunc("/api/v1/xud/tradehistory", t.GetTradeHistory).Queries("limit", "{limit}").Methods("GET")
	r.HandleFunc("/api/v1/xud/tradehistory", t.GetTradeHistory).Methods("GET")
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
