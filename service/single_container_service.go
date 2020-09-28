package service

import (
	"context"
	"fmt"
	"io"
	"strings"
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
		AbstractService: NewAbstractService(name),
		containerName:   containerName,
	}
}

func (t *SingleContainerService) SetDockerClientFactory(factory DockerClientFactory) {
	t.dockerClientFactory = factory
}

func (t *SingleContainerService) GetDockerClientFactory() DockerClientFactory {
	return t.dockerClientFactory
}

func (t *SingleContainerService) GetContainer() (*Container, error) {
	ctx := context.Background()
	cli := t.dockerClientFactory.GetSharedInstance()
	c, err := cli.ContainerInspect(ctx, t.containerName)
	if err != nil {
		return nil, err
	}
	return &Container{
		c:      &c,
		client: cli,
		logger: t.GetLogger(),
	}, nil
}

// GetStatus implements Service interface
func (t *SingleContainerService) GetStatus() (string, error) {
	status, err := t.GetContainerStatus()
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return "Container missing", nil
		}
		return "", err
	}
	return fmt.Sprintf("Container %s", status), nil
}

// GetContainerStatus is a shortcut function
func (t *SingleContainerService) GetContainerStatus() (string, error) {
	c, err := t.GetContainer()
	if err != nil {
		return "", err
	}
	return c.GetStatus(), nil
}

// GetContainerLog is a shortcut function
func (t *SingleContainerService) GetLogs(since string, tail string) (<-chan string, error) {
	c, err := t.GetContainer()
	if err != nil {
		return nil, err
	}
	return c.GetLogs(since, tail)
}

// GetContainerEnvironmentVariable is a shortcut function
func (t *SingleContainerService) GetEnvironmentVariable(key string) (string, error) {
	c, err := t.GetContainer()
	if err != nil {
		return "", err
	}
	return c.GetEnvironmentVariable(key), nil
}

// ContainerExec is a shortcut function
func (t *SingleContainerService) Exec1(command []string) (string, error) {
	c, err := t.GetContainer()
	if err != nil {
		return "", err
	}
	return c.Exec(command)
}

func (t *SingleContainerService) ExecInteractive(command []string) (string, io.Reader, io.Writer, error) {
	c, err := t.GetContainer()
	if err != nil {
		return "", nil, nil, err
	}
	return c.ExecInteractive(command)
}
