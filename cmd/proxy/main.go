package main

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	polling2 "github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"log"
	"net/http"
	"os"
	"strings"
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


	pt := polling2.Default
	wt := websocket.Default
	wt.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	logger.Info("Configuring SocketIO")
	server, err := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			pt,
			wt,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		logger.Infof("[SocketIO] New client connected: ID=%v, RemoteAddr=%v", s.ID(), s.RemoteAddr())
		return nil
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		logger.Errorf("[SocketIO] Client %v got an error: %v", s.ID(), e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		logger.Infof("[SocketIO] Client %v disconnected: %v", s.ID(), reason)
	})
	server.OnEvent("/", "console", func(s socketio.Conn, msg string) {
		logger.Infof("[CONSOLE] %s", msg)

		parts := strings.Split(msg, " ")

		switch parts[0] {
		case "open":
			s.Join("exec")
			openConsole("xud", server, logger, manager)
			// open console
		case "close":
			// close console
		case "resize":
			// resize console
		}
	})
	go server.Serve()
	defer server.Close()
	r.GET("/socket.io/", gin.WrapH(server))
	//r.POST("/socket.io/", gin.WrapH(server))
	r.Handle("WS", "/socket.io/", gin.WrapH(server))




	// Configuring CORS
	// - No origin allowed by default
	// - GET,POST, PUT, HEAD methods
	// - Credentials share disabled
	// - Preflight requests cached for 12 hours
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}

	r.Use(cors.New(config))

	//r.NotFoundHandler = Restful404Handler{}
	//r.MethodNotAllowedHandler = Restful405Handler{}


	logger.Info("Configuring router")
	manager.ConfigureRouter(r)


	logger.Infof("Serving at :%d", port)
	addr := fmt.Sprintf(":%d", port)
	err = r.Run(addr)
	if err != nil {
		log.Fatal(err)
	}
	//err = http.ListenAndServe(addr, r)
	//if err != nil {
	//	log.Fatal(err)
	//}
}

func openConsole(service string, server *socketio.Server, logger *logrus.Logger, manager *Manager) {
	s, err := manager.GetService(service)
	if err != nil {
		log.Fatal(err)
	}
	ss, ok := s.(*xud.XudService)
	if !ok {
		log.Fatal("Failed to convert to SingleContainerService")
	}
	c, err:= ss.SingleContainerService.GetContainer()
	if err != nil {
		log.Fatal(err)
	}
	execId, reader, writer, err := c.ExecInteractive([]string{"bash"})
	if err != nil {
		log.Fatal(err)
	}
	logger.Infof("Created execId %v", execId)

	server.OnEvent("/", "input", func(s socketio.Conn, msg string) {
		logger.Infof("[INPUT] %v", msg)
		_, err = writer.Write([]byte(msg))
		if err != nil {
			logger.Errorf("Failed to write: %v", err)
		}
	})

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				logger.Errorf("Failed to read: %v", err)
				break
			} else {
				logger.Infof("Read %d bytes: %v", n, buf[:n])
			}
			server.BroadcastToRoom("/", "exec", "output", string(buf[:n]))
		}
	}()
}
