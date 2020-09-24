package service

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
)

type SingleContainerService struct {
	*AbstractService
	containerName       string
	dockerClientFactory DockerClientFactory
}

func NewSingleContainerService(
	name string,
	containerName string,
) *SingleContainerService {
	return &SingleContainerService{
		AbstractService:     NewAbstractService(name),
		containerName:       containerName,
	}
}

func (t *SingleContainerService) SetDockerClientFactory(factory DockerClientFactory) {
	t.dockerClientFactory = factory
}

func (t *SingleContainerService) GetDockerClientFactory() DockerClientFactory {
	return t.dockerClientFactory
}

func (t *SingleContainerService) GetContainer() (*types.ContainerJSON, error) {
	ctx := context.Background()
	cli := t.dockerClientFactory.GetSharedInstance()
	c, err := cli.ContainerInspect(ctx, t.containerName)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (t *SingleContainerService) GetContainerStatus() (string, error) {
	c, err := t.GetContainer()
	if err != nil {
		return "", err
	}
	status := c.State.Status
	return status, nil
}

func (t *SingleContainerService) GetStatus() (string, error) {
	status, err := t.GetContainerStatus()
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("Container %s", status), nil
}
