package arby

import (
	"github.com/ExchangeUnion/xud-docker-api-poc/service"
	"github.com/gorilla/mux"
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

func (t *ArbyService) ConfigureRouter(r *mux.Router) {
	t.SingleContainerService.AbstractService.ConfigureRouter(r)
}
