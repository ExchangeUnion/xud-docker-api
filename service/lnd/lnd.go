package lnd

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/ini.v1"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"strings"
)

type LndService struct {
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
	chain      string
}

func (t *LndService) GetBackendNode() (string, error) {
	values, err := t.GetConfigValues("bitcoin.node")
	if err != nil {
		return "", err
	}
	return values[0], err
}

func New(name string, containerName string, chain string) *LndService {
	return &LndService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
		chain:                  chain,
	}
}

func (t *LndService) ConfigureRpc(options *service.RpcOptions) {

}

func (t *LndService) loadConfFile() (string, error) {
	cli := t.GetDockerClientFactory().GetSharedInstance()

	ctx := context.Background()

	filters := filters.NewArgs()
	filters.Add("reference", "alpine:latest")

	list, err := cli.ImageList(ctx, types.ImageListOptions{
		All: true,
		Filters: filters,
	})
	if cap(list) > 0 {
		log.Println("Found alpine image")
	} else {
		log.Println("ImagePull")
		out, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
		if err != nil {
			return "", err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, out)
		if err != nil {
			return "", err
		}
		log.Printf("ImagePull result\n%s", buf.String())
	}

	log.Println("ContainerCreate")
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"cat", "lnd.conf"},
		Tty:   false,
		WorkingDir: "/root/.lnd",
	}, &container.HostConfig{
		AutoRemove: true,
		Binds: []string{
			"/home/yy/.xud-docker/testnet/data/lndbtc:/root/.lnd:ro",
		},
	}, nil, "")
	if err != nil {
		return "", err
	}

	containerId := resp.ID

	rsp, err := cli.ContainerAttach(ctx, containerId, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Logs: true,
	})
	if err != nil {
		return "", err
	}

	log.Println("ContainerStart")
	err = cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if  err != nil {
		return "", err
	}

	log.Println("StdCopy")
	stdout := new(strings.Builder)
	stderr := new(strings.Builder)
	_, err = stdcopy.StdCopy(stdout, stderr, rsp.Reader)
	if err != nil {
		return "", err
	}

	log.Println("ContainerWait")
	exitCode, err := cli.ContainerWait(ctx, containerId)
	if err != nil {
		return "", err
	}
	log.Println(exitCode)

	return stdout.String(), nil
}

func (t *LndService) GetConfigValues(key string) ([]string, error) {
	result := []string{}
	c, err := t.GetContainer()
	if err != nil {
		return result, err
	}
	for k, v := range c.Config.Volumes {
		log.Printf("lndbtc volume %s: %v", k, v)
	}
	for _, bind := range c.HostConfig.Binds {
		log.Printf("lndbtc bind %s", bind)
	}

	conf, err := t.loadConfFile()
	log.Printf("Loaded lnd.conf\n%s", conf)

	log.Printf("Parsing lnd.conf as an INI file")
	config, err := ini.Load([]byte(conf))
	if err != nil {
		return result, err
	}

	section, err:= config.GetSection("Bitcoin")
	if err != nil {
		return result, errors.New("failed to get section `Bitcoind`: " + err.Error())
	}

	iniKey, err := section.GetKey(key)
	if err != nil {
		return result, errors.New(fmt.Sprintf("failed to get key `%s`: %s", key, err.Error()))
	}
	value := iniKey.String()
	log.Printf("Got %s=%s", key, value)

	return []string{value}, nil
}
