package main

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/launcher"
	"github.com/ExchangeUnion/xud-docker-api/logging"
	"github.com/ExchangeUnion/xud-docker-api/service"
	"github.com/docker/docker/pkg/homedir"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	logger    = initLogger()
	router    = initRouter()
	sioServer *socketio.Server

	port uint16
	tls bool
	network string
)

func initLogger() *logrus.Entry {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logging.Formatter{
	})
	logrus.SetOutput(os.Stdout)
	logger := logrus.NewEntry(logrus.StandardLogger())
	return logger
}

type MyWriter struct {
}

func (MyWriter) Write(p []byte) (int, error) {
	logger.Debugf("%s", strings.TrimSpace(string(p)))
	return len(p), nil
}

func initRouter() *gin.Engine {
	r := gin.New()

	// Configuring Gin middlewares
	//r.Use(ginlogrus.Logger(logrus.StandardLogger()))
	//r.Use(gin.LoggerWithWriter(MyWriter{}))
	r.Use(logging.LoggerOverLogrus())
	r.Use(gin.Recovery())

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"message": "not found"})
		} else {
			// redirect other non-API requests to ui/index.html to fit SPA requirements
			c.File("ui/index.html")
		}
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"message": "method not allowed"})
	})

	setupCors(r)

	return r
}

func setupCors(r gin.IRouter) {
	// Configuring CORS
	// - No origin allowed by default
	// - GET, POST, PUT, HEAD methods
	// - Credentials share disabled
	// - Preflight requests cached for 12 hours
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}

	r.Use(cors.New(config))
}

func initSioServer() {
	server, err := NewSioServer(network)
	if err != nil {
		logger.Fatal(err)
	}

	sioServer = server
	initSioConsole()

	go func() {
		err := server.Serve()
		defer func() {
			err := server.Close()
			if err != nil {
				logger.Fatalf("Failed to close socket.io server: %s", err)
			}
		}()
		if err != nil {
			logger.Fatal("Failed to start socket.io server")
		}
	}()

	router.GET("/socket.io/", gin.WrapH(server))
	router.Handle("WS", "/socket.io/", gin.WrapH(server))
}

func initLauncherWs() {
	router.GET("/launcher", gin.WrapF(launcher.WsHandler))
	router.Handle("WS", "/launcher", gin.WrapF(launcher.WsHandler))

	go launcher.StartLauncherRegistry()

	launcher.ConfigureRouter(router)
}

func initServiceManager() {
	logger.Debug("Creating service manager")
	manager, err := service.NewManager(network)
	if err != nil {
		logger.Fatalf("Failed to create service manager: %s", err)
	}
	defer func() {
		err := manager.Close()
		if err != nil {
			logger.Fatalf("Failed to close service manager: %s", err)
		}
	}()

	manager.ConfigureRouter(router)
}

func serve() error {
	var err error

	addr := fmt.Sprintf(":%d", port)
	logger.Infof("Serving at %s", addr)

	if tls {
		certFile := filepath.Join(homedir.Get(), ".proxy", "tls.crt")
		keyFile := filepath.Join(homedir.Get(), ".proxy", "tls.key")
		err = http.ListenAndServeTLS(addr, certFile, keyFile, router)
	} else {
		err = http.ListenAndServe(addr, router)
	}
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var err error

	network = os.Getenv("NETWORK")

	cmd := &cobra.Command{
		Use: "proxy",
		Short: "The API gateway of xud-docker",
	}
	cmd.PersistentFlags().Uint16VarP(&port, "port", "p", 8080, "The port to listen")
	cmd.PersistentFlags().BoolVar(&tls, "tls", false, "Enable TLS support")
	err = cmd.Execute()
	if err != nil {
		logger.Fatalf("Failed to parse command-line options: %s", err)
	}

	initSioServer()
	initLauncherWs()
	initServiceManager()

	err = serve()
	if err != nil {
		logger.Fatalf("Failed to serve: %s", err)
	}
}
