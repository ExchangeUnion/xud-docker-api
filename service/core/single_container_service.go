package core

import (
	"context"
	"errors"
	"fmt"
	docker "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"sync"
)

type SingleContainerService struct {
	*AbstractService

	containerName string
	dockerClient  *docker.Client
	container     *Container
	mutex         *sync.Mutex
}

func inspectContainer(client *docker.Client, name string, logger *logrus.Logger) (*Container, error) {
	ctx := context.Background()
	c, err := client.ContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	return &Container{
		c:      &c,
		client: client,
		logger: logger,
	}, nil
}

func NewSingleContainerService(
	name string,
	services map[string]Service,
	containerName string,
	dockerClient *docker.Client,
) *SingleContainerService {

	a := NewAbstractService(name, services)

	c, err := inspectContainer(dockerClient, containerName, a.logger)
	if err != nil {
		c = nil
	}

	s := &SingleContainerService{
		AbstractService:  a,
		containerName:    containerName,
		dockerClient:     dockerClient,
		container:        c,
		mutex:            &sync.Mutex{},
	}

	return s
}



func (t *SingleContainerService) GetContainer() *Container {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.container
}

// GetStatus implements Service interface
func (t *SingleContainerService) GetStatus() (string, error) {
	status, err := t.GetContainerStatus()
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			if t.IsDisabled() {
				return "Disabled", nil
			}
			return "Container missing", nil
		}
		return "", err
	}
	return fmt.Sprintf("Container %s", status), nil
}

// GetContainerStatus is a shortcut function
func (t *SingleContainerService) GetContainerStatus() (string, error) {
	c := t.GetContainer()
	if c == nil {
		return "", errors.New("container not found: " + t.containerName)
	}
	return c.GetStatus(), nil
}

// GetLogs is a shortcut function
func (t *SingleContainerService) GetLogs(since string, tail string) (<-chan string, error) {
	c := t.GetContainer()
	if c == nil {
		return nil, errors.New("container not found: " + t.containerName)
	}
	return c.GetLogs(since, tail, false)
}

// FollowLogs is a shortcut function
func (t *SingleContainerService) FollowLogs(since string, tail string) (<-chan string, error) {
	c := t.GetContainer()
	if c == nil {
		return nil, errors.New("container not found: " + t.containerName)
	}
	return c.GetLogs(since, tail, true)
}

// Getenv is a shortcut function
func (t *SingleContainerService) Getenv(key string) (string, error) {
	c := t.GetContainer()
	if c == nil {
		return "", errors.New("container not found: " + t.containerName)
	}
	return c.Getenv(key), nil
}

// Exec1 is a shortcut function
func (t *SingleContainerService) Exec1(command []string) (string, error) {
	c := t.GetContainer()
	if c == nil {
		return "", errors.New("container not found: " + t.containerName)
	}
	return c.Exec(command)
}

// Exec1 is a shortcut function
func (t *SingleContainerService) ExecInteractive(command []string) (string, io.Reader, io.Writer, error) {
	c := t.GetContainer()
	if c == nil {
		return "", nil, nil, errors.New("container not found: " + t.containerName)
	}
	return c.ExecInteractive(command)
}

func (t *SingleContainerService) OnEvent(type_ string) {
	var err error
	switch type_ {
	case "create":
		t.mutex.Lock()
		t.container, err = inspectContainer(t.dockerClient, t.containerName, t.logger)
		if err != nil {
			t.logger.Error("Failed to get container while CREATE event received: %s", err)
		}
		t.mutex.Unlock()
	case "start":
		t.mutex.Lock()
		t.container, err = inspectContainer(t.dockerClient, t.containerName, t.logger)
		if err != nil {
			t.logger.Error("Failed to get container while START event received: %s", err)
		}
		t.mutex.Unlock()
	case "die":
		t.mutex.Lock()
		t.container, err = inspectContainer(t.dockerClient, t.containerName, t.logger)
		if err != nil {
			t.logger.Error("Failed to get container while DIE event received: %s", err)
		}
		t.mutex.Unlock()
	case "destroy":
		t.mutex.Lock()
		t.container = nil
		t.mutex.Unlock()
	}
}

func (t *SingleContainerService) GetContainerId() string {
	c := t.GetContainer()
	if c == nil {
		return ""
	}
	return c.Unwrap().ID
}
