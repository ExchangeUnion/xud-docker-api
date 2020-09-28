package main

import (
	"errors"
	. "github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/arby"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/bitcoind"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/boltz"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/connext"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/geth"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/litecoind"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/lnd"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/webui"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Manager struct {
	network  string
	services []Service
}

type LightProviders struct {
	Testnet []string
	Mainnet []string
}

func NewManager(network string) (*Manager, error) {
	lightProviders := LightProviders{
		Testnet: []string{
			"http://eth.kilrau.com:52041",
			"http://michael1011.at:8546",
			"http://gethxudxv2k4pv5t5a5lswq2hcv3icmj3uwg7m2n2vuykiyv77legiad.onion:8546",
		},
		Mainnet: []string{
			"http://eth.kilrau.com:41007",
			"http://michael1011.at:8545",
			"http://gethxudxv2k4pv5t5a5lswq2hcv3icmj3uwg7m2n2vuykiyv77legiad.onion:8545",
		},
	}

	xudSvc := xud.New("xud", "testnet_xud_1")

	lndbtcSvc, err := lnd.New("lndbtc", "testnet_lndbtc_1", "bitcoin")
	if err != nil {
		return nil, err
	}

	lndltcSvc, err := lnd.New("lndltc", "testnet_lndltc_1", "litecoin")
	if err != nil {
		return nil, err
	}

	connextSvc := connext.New("connext", "testnet_connext_1")
	bitcoindSvc := bitcoind.New("bitcoind", "testnet_bitcoind_1", "lndbtc")
	litecoindSvc := litecoind.New("litecoind", "testnet_litecoind_1", "lndltc")
	gethSvc := geth.New("geth", "testnet_geth_1", "connext", lightProviders.Testnet)
	arbySvc := arby.New("arby", "testnet_arby_1")
	boltzSvc := boltz.New("boltz", "testnet_boltz_1")
	webuiSvc := webui.New("webui", "testnet_webui_1")

	manager := Manager{
		network: network,
		services: []Service{
			xudSvc,
			lndbtcSvc,
			lndltcSvc,
			connextSvc,
			bitcoindSvc,
			litecoindSvc,
			gethSvc,
			arbySvc,
			boltzSvc,
			webuiSvc,
		},
	}

	dockerClientFactory, err := NewClientFactory()
	if err != nil {
		return nil, err
	}

	xudSvc.SetServiceManager(&manager)
	xudSvc.SetDockerClientFactory(dockerClientFactory)
	xudRpc := RpcOptions{
		Host:    "xud",
		Port:    18886,
		TlsCert: "/root/.xud/tls.cert",
	}
	xudSvc.ConfigureRpc(&xudRpc)

	lndbtcSvc.SetServiceManager(&manager)
	lndbtcSvc.SetDockerClientFactory(dockerClientFactory)
	lndbtcRpc := RpcOptions{
		Host:    "lndbtc",
		Port:    10009,
		TlsCert: "/root/.lndbtc/tls.cert",
		Credential: MacaroonCredential{
			Readonly: "/root/.lndbtc/data/chain/bitcoin/testnet/readonly.macaroon",
		},
	}
	lndbtcSvc.ConfigureRpc(&lndbtcRpc)

	lndltcSvc.SetServiceManager(&manager)
	lndltcSvc.SetDockerClientFactory(dockerClientFactory)
	lndltcRpc := RpcOptions{
		Host:    "lndltc",
		Port:    10009,
		TlsCert: "/root/.lndltc/tls.cert",
		Credential: MacaroonCredential{
			Readonly: "/root/.lndltc/data/chain/litecoin/testnet/readonly.macaroon",
		},
	}
	lndltcSvc.ConfigureRpc(&lndltcRpc)

	connextSvc.SetServiceManager(&manager)
	connextSvc.SetDockerClientFactory(dockerClientFactory)

	bitcoindSvc.SetServiceManager(&manager)
	bitcoindSvc.SetDockerClientFactory(dockerClientFactory)
	bitcoindRpc := RpcOptions{
		Host: "bitcoind",
		Port: 18333,
		Credential: UsernamePasswordCredential{
			Username: "xu",
			Password: "xu",
		},
	}
	bitcoindSvc.ConfigureRpc(&bitcoindRpc)

	litecoindSvc.SetServiceManager(&manager)
	litecoindSvc.SetDockerClientFactory(dockerClientFactory)
	litecoindRpc := RpcOptions{
		Host: "litecoind",
		Port: 19333,
		Credential: UsernamePasswordCredential{
			Username: "xu",
			Password: "xu",
		},
	}
	litecoindSvc.ConfigureRpc(&litecoindRpc)

	gethSvc.SetServiceManager(&manager)
	gethSvc.SetDockerClientFactory(dockerClientFactory)
	gethRpc := RpcOptions{
		Host: "geth",
		Port: 8545,
	}
	gethSvc.ConfigureRpc(&gethRpc)

	arbySvc.SetServiceManager(&manager)
	arbySvc.SetDockerClientFactory(dockerClientFactory)

	boltzSvc.SetServiceManager(&manager)
	boltzSvc.SetDockerClientFactory(dockerClientFactory)

	webuiSvc.SetServiceManager(&manager)
	webuiSvc.SetDockerClientFactory(dockerClientFactory)

	return &manager, nil
}

func (t *Manager) getServices() []Service {
	return t.services
}

//func (t *Manager) GetStatus(w http.ResponseWriter, r *http.Request) {
//	// Container running?
//	// Processes running?
//	// Each process is health?
//
//	if service, ok := mux.Vars(r)["service"]; ok {
//		containerName := fmt.Sprintf("testnet_%s_1", service)
//		ctx := context.Background()
//		cli, err := client.NewEnvClient()
//		if err != nil {
//			log.Fatal(err)
//		}
//		cj, err := cli.ContainerInspect(ctx, containerName)
//		if err != nil {
//			log.Fatal(err)
//		}
//		err = json.NewEncoder(w).Encode(cj.State)
//		if err != nil {
//			utils.JsonError(w, err.Error(), http.StatusInternalServerError)
//		}
//	}
//}

func (t *Manager) GetStatus() (map[string]string, error) {
	result := map[string]string{}
	for _, svc := range t.services {
		status, err := svc.GetStatus()
		if err != nil {
			return nil, err
		}
		result[svc.GetName()] = status
	}
	return result, nil
}

func (t *Manager) GetService(name string) (Service, error) {
	for _, svc := range t.services {
		if svc.GetName() == name {
			return svc, nil
		}
	}
	return nil, errors.New("service not found: " + name)
}

type ServiceEntry struct {
	Id string `json:"id"`
	Name string `json:"name"`
}

func (t *Manager) ConfigureRouter(r *gin.Engine) {

	r.GET("/api/v1/services", func(c *gin.Context) {
		var result []ServiceEntry

		result = append(result, ServiceEntry{"xud", "XUD"})
		result = append(result, ServiceEntry{"lndbtc", "LND (Bitcoin)"})
		result = append(result, ServiceEntry{"lndltc", "LND (Litecoin)"})
		result = append(result, ServiceEntry{"connext", "Connext"})
		result = append(result, ServiceEntry{"bitcoind", "Bitcoind"})
		result = append(result, ServiceEntry{"litecoind", "Litecoind"})
		result = append(result, ServiceEntry{"geth", "Geth"})
		result = append(result, ServiceEntry{"arby", "Arby"})
		result = append(result, ServiceEntry{"boltz", "Boltz"})
		result = append(result, ServiceEntry{"webui", "Web UI"})

		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/v1/status", func(c *gin.Context) {
		status, err := t.GetStatus()
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	r.GET("/api/v1/status/:service", func(c *gin.Context) {
		service := c.Param("service")
		s, err := t.GetService(service)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusNotFound)
			return
		}
		status, err := s.GetStatus()
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	r.GET("/api/v1/logs/:service", func(c *gin.Context) {
		service := c.Param("service")
		s, err := t.GetService(service)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusNotFound)
			return
		}
		logs, err := s.GetLogs("1h", "all")
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "text/plain")
		for line := range logs {
			_, err = c.Writer.WriteString(line + "\n")
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			}
		}
	})

	for _, svc := range t.services {
		svc.ConfigureRouter(r)
	}
}

func (t *Manager) Close() {
	for _, svc := range t.services {
		svc.Close()
	}
}
