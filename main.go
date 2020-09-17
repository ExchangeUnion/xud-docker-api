package main

import (
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


func main() {
	xudRpc := XudRpc{}

	app := &cli.App {
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
		},
		Action: func(c *cli.Context) error {
			xudRpc.Host = c.String("xud.rpchost")
			xudRpc.Port = c.Int("xud.rpcport")
			xudRpc.Cert = c.String("xud.rpccert")
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
	xud := NewXudService(client)

	r.HandleFunc("/api/v1/xud/info", xud.GetInfo).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}
