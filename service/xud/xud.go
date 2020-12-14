package xud

import (
	"context"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api/config"
	"github.com/ExchangeUnion/xud-docker-api/service/core"
	docker "github.com/docker/docker/client"
	"os"
	"strings"
)

type Service struct {
	*core.SingleContainerService
	*RpcClient
}

func New(
	name string,
	services map[string]core.Service,
	containerName string,
	dockerClient *docker.Client,
	rpcConfig config.RpcConfig,
) *Service {
	base := core.NewSingleContainerService(name, services, containerName, dockerClient)

	return &Service{
		SingleContainerService: base,
		RpcClient:              NewRpcClient(rpcConfig, base),
	}
}

func (t *Service) GetStatus(ctx context.Context) string {
	status := t.SingleContainerService.GetStatus(ctx)
	if status != "Container running" {
		if ctx.Value("LauncherState") == "setup" {
			return "Waiting for sync"
		}
		return status
	}

	// container is running

	resp, err := t.GetInfo(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "xud is locked") {
			if _, err := os.Stat("/root/network/data/xud/nodekey.dat"); os.IsNotExist(err) {
				return "Wallet missing. Create with xucli create/restore."
			}
			return "Wallet locked. Unlock with xucli unlock."
		} else if strings.Contains(err.Error(), "no such file or directory, open '/root/.xud/tls.cert'") {
			return "Starting..."
		} else if strings.Contains(err.Error(), "xud is starting") {
			return "Starting..."
		}
		return fmt.Sprintf("Error: %s", err)
	}

	lndbtcStatus := resp.Lnd["BTC"].Status
	lndltcStatus := resp.Lnd["LTC"].Status
	connextStatus := resp.Connext.Status

	if lndbtcStatus == "Ready" && lndltcStatus == "Ready" && connextStatus == "Ready" {
		return "Ready"
	}

	if strings.Contains(lndbtcStatus, "has no active channels") ||
		strings.Contains(lndltcStatus, "has no active channels") ||
		strings.Contains(connextStatus, "has no active channels") {
		return "Waiting for channels"
	}

	var notReady []string
	if lndbtcStatus != "Ready" {
		notReady = append(notReady, "lndbtc")
	}
	if lndltcStatus != "Ready" {
		notReady = append(notReady, "lndltc")
	}
	if connextStatus != "Ready" {
		notReady = append(notReady, "connext")
	}

	return "Waiting for " + strings.Join(notReady, ", ")
}

func (t *Service) Close() error {
	err := t.RpcClient.Close()
	if err != nil {
		t.GetLogger().Errorf("Failed to close RPC client: %s", err)
	}
	return nil
}
