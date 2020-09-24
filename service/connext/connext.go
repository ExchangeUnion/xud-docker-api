package connext

import "github.com/ExchangeUnion/xud-docker-api-poc/service"

type ConnextService struct{
	*service.SingleContainerService
}

func New(name string, containerName string) *ConnextService {
	return &ConnextService{
		service.NewSingleContainerService(name, containerName),
	}
}



