package xud

import (
	"context"
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
)

type XudService struct {
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
	client pb.XudClient
	ctx    context.Context
	conn   *grpc.ClientConn
}

type XudRpc struct {
	Host string
	Port int
	Cert string
}

func NewXudService(xudRpc XudRpc) *XudService {
	var opts []grpc.DialOption

	//creds, err := credentials.NewClientTLSFromFile("/root/.xud/tls.cert", "localhost")
	//creds, err := credentials.NewClientTLSFromFile("/Users/yy/.xud-docker/simnet/data/xud/tls.cert", "")
	creds, err := credentials.NewClientTLSFromFile(xudRpc.Cert, "localhost")
	if err != nil {
		log.Fatal(err)
	}

	opts = append(opts, grpc.WithTransportCredentials(creds))
	opts = append(opts, grpc.WithBlock())
	//opts = append(opts, grpc.WithTimeout(time.Duration(10000)))

	//conn, err := grpc.Dial("xud:28886", opts...)
	//conn, err := grpc.Dial("127.0.0.1:28886", opts...)
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", xudRpc.Host, xudRpc.Port), opts...)
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewXudClient(conn)

	return &XudService{
		client: client,
		ctx:    context.Background(),
		conn:   conn,
	}
}

func (t *XudService) GetInfo(w http.ResponseWriter, r *http.Request) {
	req := pb.GetInfoRequest{}
	resp, err := t.client.GetInfo(t.ctx, &req)
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

func (t *XudService) GetBalance(w http.ResponseWriter, r *http.Request) {
	req := pb.GetBalanceRequest{}
	if currency, ok := mux.Vars(r)["currency"]; ok {
		req.Currency = currency
	}
	resp, err := t.client.GetBalance(t.ctx, &req)
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
	resp, err := t.client.TradeHistory(t.ctx, &req)
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

func New(name string, containerName string) *XudService {
	return &XudService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *XudService) ConfigureRpc(options *service.RpcOptions) {

}

func (t *XudService) ConfigureRouter(r *mux.Router) {
	t.SingleContainerService.ConfigureRouter(r)
	r.HandleFunc("/api/v1/xud/getinfo", t.GetInfo).Methods("GET")
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
