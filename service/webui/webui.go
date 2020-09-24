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