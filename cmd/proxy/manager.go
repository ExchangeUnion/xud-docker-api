package main

import (
	"errors"
	"fmt"
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
	network          string
	services         []Service
	optionalServices []string
}

func containerName(network string, service string) string {
	return fmt.Sprintf("%s_%s_1", network, service)
}

func NewManager(network string) (*Manager, error) {
	lightProviders := map[string][]string{
		"testnet": {
			"http://eth.kilrau.com:52041",
			"http://michael1011.at:8546",
			"http://gethxudxv2k4pv5t5a5lswq2hcv3icmj3uwg7m2n2vuykiyv77legiad.onion:8546",
		},
		"mainnet": {
			"http://eth.kilrau.com:41007",
			"http://michael1011.at:8545",
			"http://gethxudxv2k4pv5t5a5lswq2hcv3icmj3uwg7m2n2vuykiyv77legiad.onion:8545",
		},
	}

	xudSvc := xud.New("xud", containerName(network, "xud"))

	lndbtcSvc, err := lnd.New("lndbtc", containerName(network, "lndbtc"), "bitcoin")
	if err != nil {
		return nil, err
	}

	lndltcSvc, err := lnd.New("lndltc", containerName(network, "lndltc"), "litecoin")
	if err != nil {
		return nil, err
	}

	connextSvc := connext.New("connext", containerName(network, "connext"))

	bitcoindSvc := bitcoind.New("bitcoind", containerName(network, "bitcoind"), "lndbtc")
	litecoindSvc := litecoind.New("litecoind", containerName(network, "litecoind"), "lndltc")
	gethSvc := geth.New("geth", containerName(network, "geth"), "connext", lightProviders[network])

	arbySvc := arby.New("arby", containerName(network, "arby"))
	boltzSvc := boltz.New("boltz", containerName(network, "boltz"))
	webuiSvc := webui.New("webui", containerName(network, "webui"))

	var services []Service
	var optionalServices []string

	if network == "simnet" {
		services = []Service{
			lndbtcSvc,
			lndltcSvc,
			connextSvc,
			xudSvc,
			arbySvc,
			webuiSvc,
		}
		optionalServices = []string{
			"arby",
			"webui",
		}
	} else {
		services = []Service{
			bitcoindSvc,
			litecoindSvc,
			gethSvc,
			lndbtcSvc,
			lndltcSvc,
			connextSvc,
			xudSvc,
			arbySvc,
			boltzSvc,
			webuiSvc,
		}
		optionalServices = []string{
			"arby",
			"boltz",
			"webui",
		}
	}

	manager := Manager{
		network:          network,
		services:         services,
		optionalServices: optionalServices,
	}

	dockerClientFactory, err := NewClientFactory()
	if err != nil {
		return nil, err
	}

	xudSvc.SetServiceManager(&manager)
	xudSvc.SetDockerClientFactory(dockerClientFactory)
	var xudRpcPort int16
	if network == "simnet" {
		xudRpcPort = 28886
	} else if network == "testnet" {
		xudRpcPort = 18886
	} else if network == "mainnet" {
		xudRpcPort = 8886
	}
	xudRpc := RpcOptions{
		Host:    "xud",
		Port:    xudRpcPort,
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
			Readonly: fmt.Sprintf("/root/.lndbtc/data/chain/bitcoin/%s/readonly.macaroon", network),
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
			Readonly: fmt.Sprintf("/root/.lndltc/data/chain/litecoin/%s/readonly.macaroon", network),
		},
	}
	lndltcSvc.ConfigureRpc(&lndltcRpc)

	connextSvc.SetServiceManager(&manager)
	connextSvc.SetDockerClientFactory(dockerClientFactory)

	if network != "simnet" {
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
	}

	if network != "simnet" {
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
	}

	if network != "simnet" {
		gethSvc.SetServiceManager(&manager)
		gethSvc.SetDockerClientFactory(dockerClientFactory)
		gethRpc := RpcOptions{
			Host: "geth",
			Port: 8545,
		}
		gethSvc.ConfigureRpc(&gethRpc)
	}

	arbySvc.SetServiceManager(&manager)
	arbySvc.SetDockerClientFactory(dockerClientFactory)

	if network != "simnet" {
		boltzSvc.SetServiceManager(&manager)
		boltzSvc.SetDockerClientFactory(dockerClientFactory)
	}

	webuiSvc.SetServiceManager(&manager)
	webuiSvc.SetDockerClientFactory(dockerClientFactory)

	return &manager, nil
}

func (t *Manager) getServices() []Service {
	return t.services
}

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
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ServiceStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"`
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

		var result []ServiceStatus

		for _, svc := range t.services {
			result = append(result, ServiceStatus{Service: svc.GetName(), Status: status[svc.GetName()]})
		}

		c.JSON(http.StatusOK, result)
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
		c.JSON(http.StatusOK, ServiceStatus{Service: service, Status: status})
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
