package main

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/xudrpc"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type XudService struct {
	client pb.XudClient
	ctx    context.Context
}

func NewXudService(client pb.XudClient) *XudService {
	return &XudService{
		client: client,
		ctx:    context.Background(),
	}
}

func JsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(map[string]string{"message": message})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (t *XudService) GetInfo(w http.ResponseWriter, r *http.Request) {
	req := pb.GetInfoRequest{}
	resp, err := t.client.GetInfo(t.ctx, &req)
	if err != nil {
		JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{}
	err = m.Marshal(w, resp)
	if err != nil {
		JsonError(w, err.Error(), http.StatusInternalServerError)
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
		JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{}
	err = m.Marshal(w, resp)
	if err != nil {
		JsonError(w, err.Error(), http.StatusInternalServerError)
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
			JsonError(w, msg, http.StatusBadRequest)
			return
		}
		req.Limit = uint32(i)
	}
	resp, err := t.client.TradeHistory(t.ctx, &req)
	if err != nil {
		JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := jsonpb.Marshaler{}
	err = m.Marshal(w, resp)
	if err != nil {
		JsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}
