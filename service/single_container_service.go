package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
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
		if strings.Contains(err.Error(), "No such container") {
			return "Container missing", nil
		}
		return "", err
	}
	return fmt.Sprintf("Container %s", status), nil
}

func (t *SingleContainerService) ContainerExec(command []string) (string, error) {
	cli := t.dockerClientFactory.GetSharedInstance()
	ctx := context.Background()
	createResp, err := cli.ContainerExecCreate(ctx, t.containerName, types.ExecConfig{
		Cmd:          command,
		Tty:          false,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", err
	}

	execId := createResp.ID

	// ContainerExecAttach = ContainerExecStart
	attachResp, err := cli.ContainerExecAttach(ctx, execId, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", err
	}

	output := new(strings.Builder)
	_, err = stdcopy.StdCopy(output, output, attachResp.Reader)
	if err != nil {
		return "", err
	}

	inspectResp, err := cli.ContainerExecInspect(ctx, execId)
	if err != nil {
		return "", err
	}

	exitCode := inspectResp.ExitCode

	if exitCode != 0 {
		return output.String(), errors.New("non-zero exit code")
	}

	return output.String(), nil
}
