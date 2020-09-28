package boltz

import (
	"encoding/json"
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
)

type BoltzService struct {
	*service.SingleContainerService
}

type Node string

const (
	BTC Node = "btc"
	LTC Node = "ltc"
)

func New(
	name string,
	containerName string,
) *BoltzService {
	return &BoltzService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

// {
//  "symbol": "BTC",
//  "lnd_pubkey": "02c882fbd75ba7c0e3175a0b86037b4d056599a694fcfad56589fc05d081b62774",
//  "block_height": 1835961
// }
func (t *BoltzService) GetInfo(node Node) (map[string]interface{}, error) {
	output, err := t.Exec1([]string{"wrapper", string(node), "getinfo"})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type NodeStatus struct {
	Status string
	IsUp   bool
}

func (t *BoltzService) checkNode(node Node) NodeStatus {
	_, err := t.GetInfo(node)
	if err == nil {
		return NodeStatus{Status: string(node) + " up", IsUp: true}
	} else {
		return NodeStatus{Status: string(node) + " down", IsUp: false}
	}
}

func (t *BoltzService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status != "Container running" {
		return status, err
	}

	btcStatus := t.checkNode(BTC)
	ltcStatus := t.checkNode(LTC)

	if btcStatus.IsUp && ltcStatus.IsUp {
		return "Ready", nil
	} else {
		return btcStatus.Status + "; " + ltcStatus.Status, nil
	}
}
