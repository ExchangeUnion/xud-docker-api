package arby

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
)

type ArbyService struct {
	*service.SingleContainerService
}

func New(
	name string,
	containerName string,
) *ArbyService {
	return &ArbyService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *ArbyService) GetStatus() (string, error) {
	status, err := t.SingleContainerService.GetStatus()
	if err != nil {
		return "", err
	}
	if status == "Container running" {
		return "Ready", nil
	} else {
		return status, nil
	}
}
