package main

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/docker/docker/pkg/homedir"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net/http"
	"os"
	"path"
)

var (
	logger    = initLogger()
	router    = initRouter()
	sioServer *socketio.Server
)

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

func initRouter() *gin.Engine {
	r := gin.Default()
	setupCors(r)
	return r
}

type Config struct {
	Port int
}

func loadConfig() (*Config, error) {
	logger.Debug("Loading config")

	err := godotenv.Load("/root/config.sh")
	if err != nil {
		logger.Debug("Skip /root/config.sh")
	}

	var port int

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "port, p",
				Value: 8080,
			},
		},
		Action: func(c *cli.Context) error {
			port = c.Int("port")
			return nil
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}

	config := Config{Port: port}

	return &config, nil
}

func setupCors(r gin.IRouter) {
	// Configuring CORS
	// - No origin allowed by default
	// - GET, POST, PUT, HEAD methods
	// - Credentials share disabled
	// - Preflight requests cached for 12 hours
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://localhost:3000"}

	r.Use(cors.New(config))
}

func main() {

	config, err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %s", err)
	}

	network := os.Getenv("NETWORK")

	logger.Debug("Creating service manager")
	manager, err := service.NewManager(network)
	if err != nil {
		logger.Fatalf("Failed to create service manager: %s", err)
	}
	defer manager.Close()

	server, err := NewSioServer(network)
	sioServer = server
	initSioConsole()

	if err != nil {
		logger.Fatal(err)
	}

	go func() {
		err := server.Serve()
		defer server.Close()
		if err != nil {
			logger.Fatal("Failed to start socket.io server")
		}
	}()

	logger.Debug("Creating router")

	r := router

	r.GET("/socket.io/", gin.WrapH(server))
	r.Handle("WS", "/socket.io/", gin.WrapH(server))

	r.NoRoute(func(c *gin.Context) {
		c.File("ui/index.html")
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"message": "method not allowed"})
	})

	logger.Debug("Configuring router")
	manager.ConfigureRouter(r)

	logger.Infof("Serving at :%d", config.Port)
	addr := fmt.Sprintf(":%d", config.Port)

	certFile := path.Join(homedir.Get(), ".proxy", "tls.crt")
	keyFile := path.Join(homedir.Get(), ".proxy", "tls.key")

	err = http.ListenAndServeTLS(addr, certFile, keyFile, r)
	//err = http.ListenAndServe(addr, r)
	if err != nil {
		logger.Fatalf("Failed to start the server: %s", err)
	}
}
