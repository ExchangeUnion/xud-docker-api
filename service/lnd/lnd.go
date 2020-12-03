package lnd

import (
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/config"
	"github.com/ExchangeUnion/xud-docker-api-poc/service/core"
	docker "github.com/docker/docker/client"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"strings"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient

	chain      string
	logWatcher *LogWatcher
}

func (t *Service) GetBackendNode() (string, error) {
	key := fmt.Sprintf("%s.node", t.chain)
	values, err := t.GetConfigValues(key)
	if err != nil {
		return "", err
	}
	return values[0], err
}

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	chain string,
	rpcConfig config.RpcConfig,
) *Service {

	base := core.NewSingleContainerService(name, services, containerName, dockerClient)

	w := NewLogWatcher(containerName, base.GetLogger().WithField("scope", "LogWatcher"), base)

	s := &Service{
		SingleContainerService: base,
		RpcClient:              NewRpcClient(rpcConfig, base.GetLogger().WithField("scope", "RPC")),
		chain:                  chain,
		logWatcher:             w,
	}

	go w.Start()

	return s
}

func (t *Service) loadConfFile() (string, error) {
	confFile := fmt.Sprintf("/root/network/data/%s/lnd.conf", t.GetName())
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (t *Service) GetConfigValues(key string) ([]string, error) {
	var result []string
	//c, err := t.GetContainer()
	//if err != nil {
	//	return result, err
	//}
	//for k, v := range c.Config.Volumes {
	//	log.Printf("lndbtc volume %s: %v", k, v)
	//}
	//for _, bind := range c.HostConfig.Binds {
	//	log.Printf("lndbtc bind %s", bind)
	//}

	conf, err := t.loadConfFile()

	config, err := ini.ShadowLoad([]byte(conf))
	if err != nil {
		return result, err
	}

	parts := strings.Split(key, ".")

	if cap(parts) == 2 {
		section, err := config.GetSection(strings.Title(parts[0]))
		if err != nil {
			return result, err
		}

		iniKey, err := section.GetKey(key)
		if err != nil {
			return result, err
		}
		value := iniKey.Value()
		result = append(result, value)
	} else if cap(parts) == 1 {
		section, err := config.GetSection(ini.DefaultSection)
		if err != nil {
			return result, err
		}

		iniKey, err := section.GetKey(key)
		if err != nil {
			return result, err
		}
		values := iniKey.ValueWithShadows()
		result = append(result, values...)
	}

	return result, nil
}

func (t *Service) Neutrino() bool {
	// TODO get lnd backend type
	return true
}

func syncingText(current int64, total int64) string {
	if total < current {
		total = current
	}
	p := float32(current) / float32(total) * 100.0
	if p > 0.005 {
		p = p - 0.005
	} else {
		p = 0
	}
	return fmt.Sprintf("Syncing %.2f%% (%d/%d)", p, current, total)
}

func (t *Service) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		info, err := t.GetInfo()
		if err != nil {
			if strings.Contains(err.Error(), "Wallet is encrypted") {
				return "Wallet locked. Unlock with lncli unlock.", nil
			} else if strings.Contains(err.Error(), "no such file or directory") {
				if t.Neutrino() {
					return t.logWatcher.GetNeutrinoStatus(), nil
				}
			} else if strings.Contains(err.Error(), "no client") {
				if t.Neutrino() {
					return t.logWatcher.GetNeutrinoStatus(), nil
				}
			}
			return "", err
		}

		syncedToChain := info.SyncedToChain
		total := info.BlockHeight
		current, err := t.logWatcher.getCurrentHeight()

		//t.GetLogger().Infof("Current height is %d", current)

		if err == nil && current > 0 {
			if total <= current {
				return "Ready", nil
			} else {
				return syncingText(int64(current), int64(total)), nil
			}
		} else {
			if syncedToChain {
				return "Ready", nil
			} else {
				return "Syncing", nil
			}
		}
	} else {
		return status, nil
	}
}

func (t *Service) Close() error {
	_ = t.RpcClient.Close()
	return nil
}
