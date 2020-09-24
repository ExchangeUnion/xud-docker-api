package boltz

import "github.com/ExchangeUnion/xud-docker-api-poc/service"

type BoltzService struct{
	*service.SingleContainerService
}

func New(
	name string,
	containerName string,
) *BoltzService {
	return &BoltzService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}
