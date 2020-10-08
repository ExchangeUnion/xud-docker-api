package webui

import "github.com/ExchangeUnion/xud-docker-api-poc/service"

type WebuiService struct {
	*service.SingleContainerService
}

func (t *WebuiService) GetName() string {
	return "webui"
}

func New(
	name string,
	containerName string,
) *WebuiService {
	return &WebuiService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *WebuiService) GetStatus() (string, error) {
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
