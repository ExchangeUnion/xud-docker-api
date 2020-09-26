package lnd

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/lnd/lnrpc"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/ini.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type LndService struct {
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
	rpcClient  pb.LightningClient
	chain      string
	p          *regexp.Regexp
}

func (t *LndService) GetBackendNode() (string, error) {
	values, err := t.GetConfigValues(fmt.Sprintf("%s.node", t.chain))
	if err != nil {
		return "", err
	}
	return values[0], err
}

func New(name string, containerName string, chain string) (*LndService, error) {
	p, err := regexp.Compile("^.*NTFN: New block: height=(\\d+), sha=(.+)$")
	if err != nil {
		return nil, err
	}

	return &LndService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
		chain:                  chain,
		p:                      p,
	}, nil
}

func (t *LndService) ConfigureRpc(options *service.RpcOptions) {
	t.rpcOptions = options
}

func (t *LndService) getRpcClient() (pb.LightningClient, error) {
	if t.rpcClient == nil {
		creds, err := credentials.NewClientTLSFromFile(t.rpcOptions.TlsCert, "localhost")
		if err != nil {
			return nil, err
		}

		addr := fmt.Sprintf("%s:%d", t.rpcOptions.Host, t.rpcOptions.Port)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithBlock())
		//opts = append(opts, grpc.WithTimeout(time.Duration(10000)))

		macaroonCred, ok := t.rpcOptions.Credential.(service.MacaroonCredential)
		if !ok {
			return nil, errors.New("MacaroonCredential is required")
		}

		opts = append(opts, grpc.WithPerRPCCredentials(macaroonCred))

		conn, err := grpc.Dial(addr, opts...)
		if err != nil {
			return nil, err
		}

		t.rpcClient = pb.NewLightningClient(conn)
	}
	return t.rpcClient, nil
}

func (t *LndService) loadConfFileFallback() (string, error) {
	cli := t.GetDockerClientFactory().GetSharedInstance()

	ctx := context.Background()

	filters := filters.NewArgs()
	filters.Add("reference", "alpine:latest")

	list, err := cli.ImageList(ctx, types.ImageListOptions{
		All:     true,
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
		Image:      "alpine",
		Cmd:        []string{"cat", "lnd.conf"},
		Tty:        false,
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
		Logs:   true,
	})
	if err != nil {
		return "", err
	}

	log.Println("ContainerStart")
	err = cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
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

func (t *LndService) loadConfFile() (string, error) {
	confFile := fmt.Sprintf("/root/.%s/lnd.conf", t.GetName())
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (t *LndService) GetConfigValues(key string) ([]string, error) {
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
	log.Printf("Loaded lnd.conf\n%s", conf)

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

func (t *LndService) GetInfo() (*pb.GetInfoResponse, error) {
	client, err := t.getRpcClient()
	if err != nil {
		return nil, err
	}

	req := pb.GetInfoRequest{}

	return client.GetInfo(context.Background(), &req)
}

func (t *LndService) ConfigureRouter(r *gin.Engine) {
	r.GET(fmt.Sprintf("/api/v1/%s/getinfo", t.GetName()), func(c *gin.Context) {
		resp, err := t.GetInfo()
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		m := jsonpb.Marshaler{EmitDefaults: true}
		err = m.Marshal(c.Writer, resp)
		if err != nil {
			utils.JsonError(c, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
	})
}

func (t *LndService) getCurrentHeight() (uint32, error) {
	logs, err := t.GetLogs("10m", "all")
	if err != nil {
		return 0, nil
	}

	var height string

	for line := range logs {
		if t.p.MatchString(line) {
			height = t.p.ReplaceAllString(line, "$1")
		}
	}

	if height != "" {
		i64, err := strconv.ParseInt(height, 10, 32)
		if err != nil {
			return 0, nil
		}
		return uint32(i64), nil
	}

	return 0, nil
}

func (t *LndService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		info, err := t.GetInfo()
		if err != nil {
			if strings.Contains(err.Error(), "Wallet is encrypted") {
				return "Wallet locked. Unlock with xucli unlock.", nil
			}
			return "", err
		}

		syncedToChain := info.SyncedToChain
		total := info.BlockHeight
		current, err := t.getCurrentHeight()

		t.GetLogger().Infof("Current height is %d", current)

		if err == nil && current > 0 {
			if total <= current {
				return "Ready", nil
			} else {
				p := float32(current) / float32(total) * 100.0
				if p > 0.005 {
					p = p - 0.005
				} else {
					p = 0
				}
				return fmt.Sprintf("Syncing %.2f%% (%d/%d)", p, current, total), nil
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
