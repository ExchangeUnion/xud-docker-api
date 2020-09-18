package main

import (
	"context"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/xudrpc"
	"github.com/golang/protobuf/jsonpb"
	"net/http"
)

type XudService struct {
	client pb.XudClient
}

func NewXudService(client pb.XudClient) *XudService {
	return &XudService{
		client: client,
	}
}

func (t *XudService) GetInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	req := pb.GetInfoRequest{}
	resp, err := t.client.GetInfo(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	m := jsonpb.Marshaler{}
	w.Header().Set("Content-Type", "application/json")
	err = m.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

