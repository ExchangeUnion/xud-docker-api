package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/arby"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/bitcoind"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/boltz"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/connext"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/geth"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/litecoind"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/lnd"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/webui"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/xud"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var (
	lightProviders = map[string][]string{
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
)

type Manager struct {
	network   string
	services  []core.Service
	factory   core.DockerClientFactory
	logger    *logrus.Logger
	listeners map[string]core.Listener
}

func containerName(network string, service string) string {
	return fmt.Sprintf("%s_%s_1", network, service)
}

func initServices(network string, dockerClient *docker.Client, listeners map[string]core.Listener) []core.Service {

	f, err := ioutil.ReadFile("/root/network/data/config.json")
	if err != nil {
		panic(err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(f, &config)
	if err != nil {
		panic(err)
	}

	j := config["services"].([]map[string]interface{})

	var result []core.Service
	var resultMap = make(map[string]core.Service)
	var s core.Service
	var name string
	var cName string
	var rpc map[string]interface{}
	var disabled bool
	var mode string

	for _, x := range j {
		name = x["name"].(string)
		cName = containerName(network, name)
		rpc = x["rpc"].(map[string]interface{})
		disabled = x["disabled"].(bool)
		if x["mode"] == nil {
			mode = ""
		} else {
			mode = x["mode"].(string)
		}
		switch name {
		case "bitcoind":
			s = bitcoind.New(name, resultMap, cName, dockerClient, "lndbtc", rpc)
		case "litecoind":
			s = litecoind.New(name, resultMap, cName, dockerClient, "lndltc", rpc)
		case "geth":
			s = geth.New(name, resultMap, cName, dockerClient, "connext", lightProviders[network], rpc)
		case "lndbtc":
			s = lnd.New(name, resultMap, cName, dockerClient, "bitcoin", rpc)
		case "lndltc":
			s = lnd.New(name, resultMap, cName, dockerClient, "litecoin", rpc)
		case "connext":
			s = connext.New(name, resultMap, cName, dockerClient, rpc)
		case "xud":
			s = xud.New(name, resultMap, cName, dockerClient, rpc)
		case "arby":
			s = arby.New(name, resultMap, cName, dockerClient, rpc)
		case "boltz":
			s = boltz.New(name, resultMap, cName, dockerClient, rpc)
		case "webui":
			s = webui.New(name, resultMap, cName, dockerClient)
		default:
			panic(errors.New("unsupported service: " + name))
		}

		s.SetDisabled(disabled)
		s.SetMode(mode)

		result = append(result, s)
		resultMap[s.GetName()] = s

		listeners[cName] = s
	}

	// add self
	s = webui.New("proxy", resultMap, containerName(network, "proxy"), dockerClient)
	result = append(result, s)
	resultMap[s.GetName()] = s

	return result
}

func NewManager(network string) (*Manager, error) {
	factory, err := core.NewClientFactory()
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	listeners := map[string]core.Listener{}

	manager := Manager{
		network:   network,
		services:  initServices(network, factory.GetSharedInstance(), listeners),
		factory:   factory,
		logger:    logger,
		listeners: listeners,
	}

	go manager.listenForDockerEvents()

	return &manager, nil
}

func (t *Manager) getServices() []core.Service {
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
			t.logger.Debugf("[Status] %s: %s", s.GetName(), status)
			ch <- StatusResult{Service: s.GetName(), Status: status}
		}()
	}

	t.logger.Debugf("Services count: %d %v %d", cap(t.services), t.services, len(t.services))

	for i := 0; i < len(t.services); i++ {
		r := <-ch
		result[r.Service] = r.Status
	}

	return result
}

func (t *Manager) GetService(name string) (core.Service, error) {
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

func (t *Manager) Close() {
	for _, s := range t.services {
		s.Close()
	}
}

func (t *Manager) id2name(id string) string {
	client := t.factory.GetSharedInstance()
	ctx := context.Background()
	c, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return ""
	}
	// the container name is started with "/"
	return c.Name[1:]
}

func (t *Manager) listenForDockerEvents() {
	client := t.factory.GetSharedInstance()
	events, errs := client.Events(context.Background(), types.EventsOptions{})

	var name string
	t.logger.Debug("Starting listening for Docker events")
ListenLoop:
	for {
		select {
		case event := <-events:
			//t.logger.Debugf("Got Docker event: %v", event)
			// TODO set hasContainer to true
			if event.Type == "container" {
				switch event.Action {
				case "create":
					name = t.id2name(event.ID)
					t.logger.Debugf("[Event] Container create: %s", name)
					s, ok := t.listeners[name]
					if ok {
						s.OnEvent("create")
					}
				case "start":
					name = t.id2name(event.ID)
					t.logger.Debugf("[Event] Container start: %s", name)
					s, ok := t.listeners[name]
					if ok {
						s.OnEvent("start")
					}
				case "die":
					name = t.id2name(event.ID)
					t.logger.Debugf("[Event] Container die: %s", name)
					s, ok := t.listeners[name]
					if ok {
						s.OnEvent("die")
					}
				case "destroy":
					t.logger.Debugf("[Event] Container destroy: %s", event.ID)
					for _, s := range t.services {
						if s.GetContainerId() == event.ID {
							s.OnEvent("die")
							break
						}
					}
				}
			}
		case err := <-errs:
			t.logger.Debugf("Got an error while listening for Docker events: %v", err)
			break ListenLoop
		}
	}
	t.logger.Debug("Stopped listening for Docker events")
}
