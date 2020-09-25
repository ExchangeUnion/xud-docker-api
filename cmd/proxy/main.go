package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

type Restful404Handler struct{}
type Restful405Handler struct{}

//func (Restful404Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	w.Header().Set("Content-Type", "application/json; charset=utf-8")
//	w.Header().Set("X-Content-Type-Options", "nosniff")
//	w.WriteHeader(http.StatusNotFound)
//	err := json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//	}
//}
//
//func (Restful405Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	w.Header().Set("Content-Type", "application/json; charset=utf-8")
//	w.Header().Set("X-Content-Type-Options", "nosniff")
//	w.WriteHeader(http.StatusMethodNotAllowed)
//	err := json.NewEncoder(w).Encode(map[string]string{"message": "method not allowed"})
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//	}
//}

func main() {
	logger := logrus.New()

	var port int
	//xudRpc := XudRpc{}

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
			//xudRpc.Host = c.String("xud.rpchost")
			//xudRpc.Port = c.Int("xud.rpcport")
			//xudRpc.Cert = c.String("xud.rpccert")
			port = c.Int("port")
			return nil
		},
	}

	logger.Info("Parsing command-line arguments")
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Creating service manager")
	manager, err := NewManager("testnet")
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	logger.Info("Creating router")
	r := gin.Default()
	//r.NotFoundHandler = Restful404Handler{}
	//r.MethodNotAllowedHandler = Restful405Handler{}

	logger.Info("Configuring router")
	manager.ConfigureRouter(r)

	logger.Infof("Starting server on :%d", port)
	addr := fmt.Sprintf(":%d", port)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}
