package main

import (
	"encoding/json"
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
	"github.com/ExchangeUnion/xud-docker-api-poc/service/proxy"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/webui"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/hpcloud/tail"
	"io"
	"net/http"
	"strings"
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
	proxySvc := proxy.New("proxy", containerName(network, "proxy"))

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
			proxySvc,
		}
		optionalServices = []string{
			"arby",
			"webui",
			"proxy",
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
			proxySvc,
		}
		optionalServices = []string{
			"arby",
			"boltz",
			"webui",
			"proxy",
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
	var bitcoindRpcPort int16
	var litecoindRpcPort int16

	if network == "simnet" {
		xudRpcPort = 28886
		bitcoindRpcPort = 28332
		litecoindRpcPort = 29332
	} else if network == "testnet" {
		xudRpcPort = 18886
		bitcoindRpcPort = 18332
		litecoindRpcPort = 19332
	} else if network == "mainnet" {
		xudRpcPort = 8886
		bitcoindRpcPort = 8332
		litecoindRpcPort = 9332
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
			Port: bitcoindRpcPort,
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
			Port: litecoindRpcPort,
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

	proxySvc.SetServiceManager(&manager)
	proxySvc.SetDockerClientFactory(dockerClientFactory)

	return &manager, nil
}

func (t *Manager) getServices() []Service {
	return t.services
}

type StatusResult struct {
	Service string
	Status  string
}

func (t *Manager) GetStatus() map[string]string {
	result := map[string]string{}
	ch := make(chan StatusResult)
	for _, svc := range t.services {
		s := svc
		go func() {
			status, err := s.GetStatus()
			if err != nil {
				status = fmt.Sprintf("Error: %s", err)
			}
			ch <- StatusResult{Service: s.GetName(), Status: status}
		}()
	}

	for i := 0; i < cap(t.services); i++ {
		r := <-ch
		result[r.Service] = r.Status
	}

	return result
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

type SetupStatus struct {
	Status  string      `json:"status"`
	Details interface{} `json:"details"`
}

func (t *Manager) ConfigureRouter(r *gin.Engine) {
	r.Use(static.Serve("/", static.LocalFile("/ui", false)))

	api := r.Group("/api")
	{
		api.GET("/v1/services", func(c *gin.Context) {
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

		api.GET("/v1/status", func(c *gin.Context) {
			status := t.GetStatus()

			var result []ServiceStatus

			for _, svc := range t.services {
				result = append(result, ServiceStatus{Service: svc.GetName(), Status: status[svc.GetName()]})
			}

			c.JSON(http.StatusOK, result)
		})

		api.GET("/v1/status/:service", func(c *gin.Context) {
			service := c.Param("service")
			s, err := t.GetService(service)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusNotFound)
				return
			}
			status, err := s.GetStatus()
			if err != nil {
				status = fmt.Sprintf("Error: %s", err)
			}
			c.JSON(http.StatusOK, ServiceStatus{Service: service, Status: status})
		})

		api.GET("/v1/logs/:service", func(c *gin.Context) {
			service := c.Param("service")
			s, err := t.GetService(service)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusNotFound)
				return
			}
			since := c.DefaultQuery("since", "1h")
			tail := c.DefaultQuery("tail", "all")
			logs, err := s.GetLogs(since, tail)
			if err != nil {
				utils.JsonError(c, err.Error(), http.StatusInternalServerError)
				return
			}
			c.Header("Content-Type", "text/plain")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.log\"", service))
			for line := range logs {
				_, err = c.Writer.WriteString(line + "\n")
				if err != nil {
					utils.JsonError(c, err.Error(), http.StatusInternalServerError)
				}
			}
		})

		api.GET("/v1/setup-status", func(c *gin.Context) {
			c.Stream(func(w io.Writer) bool {
				logfile := fmt.Sprintf("/root/network/logs/%s.log", t.network)
				t, err := tail.TailFile(logfile, tail.Config{
					Follow: true,
					ReOpen: true})
				if err != nil {
					return false
				}
				for line := range t.Lines {
					if strings.Contains(line.Text, "Waiting for XUD dependencies to be ready") {
						status := SetupStatus{Status: "Waiting for XUD dependencies to be ready", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "LightSync") {
						parts := strings.Split(line.Text, " [LightSync] ")
						parts = strings.Split(parts[1], " | ")
						details := map[string]string{}
						status := SetupStatus{Status: "Syncing light clients", Details: details}
						for _, p := range parts {
							kv := strings.Split(p, ": ")
							details[kv[0]] = kv[1]
						}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Setup wallets") {
						status := SetupStatus{Status: "Setup wallets", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Create wallets") {
						status := SetupStatus{Status: "Create wallets", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Restore wallets") {
						status := SetupStatus{Status: "Restore wallets", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Setup backup location") {
						status := SetupStatus{Status: "Setup backup location", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Unlock wallets") {
						status := SetupStatus{Status: "Unlock wallets", Details: nil}
						j, _ := json.Marshal(status)
						c.Writer.Write(j)
						c.Writer.Write([]byte("\n"))
						c.Writer.Flush()
					} else if strings.Contains(line.Text, "Start shell") {
						break
					}
				}
				return false
			})
		})

	}

	for _, svc := range t.services {
		svc.ConfigureRouter(api)
	}
}

func (t *Manager) Close() {
	for _, svc := range t.services {
		svc.Close()
	}
}
