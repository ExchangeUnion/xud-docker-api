package main

import (
	"encoding/json"
	"fmt"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/xudrpc"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

type XudRpc struct {
	Host string
	Port int
	Cert string
}

type Restful404Handler struct{}
type Restful405Handler struct{}

func (Restful404Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	err := json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (Restful405Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusMethodNotAllowed)
	err := json.NewEncoder(w).Encode(map[string]string{"message": "method not allowed"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	var port int
	xudRpc := XudRpc{}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "xud.rpchost",
			},
			&cli.IntFlag{
				Name: "xud.rpcport",
			},
			&cli.StringFlag{
				Name: "xud.rpccert",
			},
			&cli.IntFlag{
				Name:  "port, p",
				Value: 8080,
			},
		},
		Action: func(c *cli.Context) error {
			xudRpc.Host = c.String("xud.rpchost")
			xudRpc.Port = c.Int("xud.rpcport")
			xudRpc.Cert = c.String("xud.rpccert")
			port = c.Int("port")
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

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
	defer conn.Close()

	client := pb.NewXudClient(conn)

	r := mux.NewRouter()
	r.NotFoundHandler = Restful404Handler{}
	r.MethodNotAllowedHandler = Restful405Handler{}

	xud := NewXudService(client)

	r.HandleFunc("/api/v1/xud/getinfo", xud.GetInfo).Methods("GET")
	r.HandleFunc("/api/v1/xud/getbalance", xud.GetBalance).Methods("GET")
	r.HandleFunc("/api/v1/xud/getbalance/{currency}", xud.GetBalance).Methods("GET")
	r.HandleFunc("/api/v1/xud/tradehistory", xud.GetTradeHistory).Methods("GET")
	r.HandleFunc("/api/v1/xud/tradehistory", xud.GetTradeHistory).Queries("limit", "{limit}").Methods("GET")

	addr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(addr, r))
}
