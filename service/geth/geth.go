package geth

import "github.com/ExchangeUnion/xud-docker-api-poc/service"

type GethService struct{
	*service.SingleContainerService
	rpcOptions *service.RpcOptions
}

func New(name string, containerName string) *GethService {
	return &GethService{
		SingleContainerService: service.NewSingleContainerService(name, containerName),
	}
}

func (t *GethService) ConfigureRpc(options *service.RpcOptions) {

}

