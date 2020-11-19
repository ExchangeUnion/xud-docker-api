package lnd

import (
	"context"
	"errors"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	pb "github.com/ExchangeUnion/xud-docker-api-poc/service/lnd/lnrpc"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NeutrinoSyncing struct {
	current int64
	total   int64
	done    bool
}

type LndService struct {
	*service.SingleContainerService
	rpcOptions      *service.RpcOptions
	rpcClient       pb.LightningClient
	chain           string
	p               *regexp.Regexp
	p0              *regexp.Regexp
	p1              *regexp.Regexp
	p2              *regexp.Regexp
	neutrinoSyncing NeutrinoSyncing
}

func (t *LndService) GetBackendNode() (string, error) {
	key := fmt.Sprintf("%s.node", t.chain)
	values, err := t.GetConfigValues(key)
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

	p0, err := regexp.Compile("^.*Fully caught up with cfheaders at height (\\d+), waiting at tip for new blocks$")
	if err != nil {
		return nil, err
	}

	var p1 *regexp.Regexp

	if strings.Contains(containerName, "simnet") {
		p1, err = regexp.Compile("^.*Writing cfheaders at height=(\\d+) to next checkpoint$")
		if err != nil {
			return nil, err
		}
	} else {
		p1, err = regexp.Compile("^.*Fetching set of checkpointed cfheaders filters from height=(\\d+).*$")
		if err != nil {
			return nil, err
		}
	}

	p2, err := regexp.Compile("^.*Syncing to block height (\\d+) from peer.*$")
	if err != nil {
		return nil, err
	}

	s := LndService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
		chain:                  chain,
		p:                      p,
		p0:                     p0,
		p1:                     p1,
		p2:                     p2,
		neutrinoSyncing:        NeutrinoSyncing{current: 0, total: 0, done: false},
	}

	go func() {
		err := s.watchNeutrinoSyncing()
		if err != nil {
			s.GetLogger().Error("Failed to watch Neutrino syncing", err)
		}
	}()

	return &s, nil
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

		macaroonCred, ok := t.rpcOptions.Credential.(service.MacaroonCredential)
		if !ok {
			return nil, errors.New("MacaroonCredential is required")
		}

		opts = append(opts, grpc.WithPerRPCCredentials(macaroonCred))

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			if err.Error() == "context deadline exceeded" {
				return nil, errors.New("cannot establish gRPC connection")
			}
			return nil, err
		}

		t.rpcClient = pb.NewLightningClient(conn)
	}
	return t.rpcClient, nil
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

func (t *LndService) ConfigureRouter(r *gin.RouterGroup) {
	r.GET(fmt.Sprintf("/v1/%s/getinfo", t.GetName()), func(c *gin.Context) {
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

func (t *LndService) watchNeutrinoSyncing() error {
	t.GetLogger().Debug("[watch] Neutrino syncing")
	c, err := t.GetContainer()
	for err != nil && c != nil {
		t.GetLogger().Debug("[watch] Waiting for Docker container to be created")
		time.Sleep(1 * time.Second)
		c, err = t.GetContainer()
	}
	t.GetLogger().Debug("[watch] Got container")
	startedAt := c.Unwrap().State.StartedAt
	t.GetLogger().Debugf("[watch] startedAt=%s", startedAt)
	logs, err := t.FollowLogs("1h", "")
	if err != nil {
		return err
	}
	t.GetLogger().Debug("[watch] Watch logs")
	for line := range logs {

		line = strings.TrimSpace(line)
		var current string
		var total string

		if t.p0.MatchString(line) {
			t.GetLogger().Debugf("[watch] <p0> %s", line)
			current = t.p0.ReplaceAllString(line, "$1")
			t.neutrinoSyncing.current, err = strconv.ParseInt(current, 10, 64)
			if err != nil {
				return err
			}
			t.neutrinoSyncing.done = true
		} else {
			if t.p1.MatchString(line) {
				t.GetLogger().Debugf("[watch] <p1> %s", line)
				current = t.p1.ReplaceAllString(line, "$1")
				t.neutrinoSyncing.current, err = strconv.ParseInt(current, 10, 64)
				if err != nil {
					return err
				}
			} else {
				if t.p2.MatchString(line) {
					t.GetLogger().Debugf("[watch] <p2> %s", line)
					total = t.p2.ReplaceAllString(line, "$1")
					t.neutrinoSyncing.total, err = strconv.ParseInt(total, 10, 64)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	t.GetLogger().Debug("[watch] Done")

	return nil
}

func (t *LndService) Neutrino() bool {
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

func (t *LndService) GetNeutrinoStatus() string {
	current := t.neutrinoSyncing.current
	total := t.neutrinoSyncing.total
	return syncingText(current, total)
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
				return "Wallet locked. Unlock with lncli unlock.", nil
			}
			if strings.Contains(err.Error(), "no such file or directory") {
				if t.Neutrino() {
					return t.GetNeutrinoStatus(), nil
				}
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
