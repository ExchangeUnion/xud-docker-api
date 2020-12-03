package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"strings"
	"sync"
)

type SingleContainerService struct {
	*AbstractService

	containerName string
	dockerClient  *docker.Client
	mutex         *sync.Mutex
	container     *types.ContainerJSON
	cond  *sync.Cond
}

func NewSingleContainerService(
	name string,
	services map[string]Service,
	containerName string,
	dockerClient *docker.Client,
) *SingleContainerService {

	mutex := &sync.Mutex{}

	s := &SingleContainerService{
		AbstractService: NewAbstractService(name, services),
		containerName:   containerName,
		dockerClient:    dockerClient,
		mutex:           mutex,
		container:       nil,
		cond:    sync.NewCond(mutex),
	}

	go s.initContainer()

	return s
}

func (t *SingleContainerService) getContainer() (*types.ContainerJSON, error) {
	c, err := t.dockerClient.ContainerInspect(context.Background(), t.containerName)
	if err != nil {
		return nil, err
	}
	return &c, err
}

// GetStatus implements Service interface
func (t *SingleContainerService) GetStatus() (string, error) {
	status, err := t.GetContainerStatus()
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			if t.IsDisabled() && (t.GetMode() == "" || t.GetMode() == "native") {
				return "Disabled", nil
			}
			return "Container missing", nil
		}
		return "", err
	}
	return fmt.Sprintf("Container %s", status), nil
}

func (t *SingleContainerService) GetContainerStatus() (string, error) {
	c, err := t.getContainer()
	if err != nil {
		return "", err
	}
	return c.State.Status, nil
}

func (t *SingleContainerService) GetContainerId() string {
	c, err := t.getContainer()
	if err != nil {
		t.logger.Debugf("Failed to get container %s ID: %s", t.containerName, err)
		return ""
	}
	return c.ID
}

func (t *SingleContainerService) getLogs(since string, tail string, follow bool) (<-chan string, error) {
	reader, err := t.dockerClient.ContainerLogs(context.Background(), t.containerName, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since,
		Tail:       tail,
		Follow:     follow,
	})

	if err != nil {
		return nil, err
	}

	ch := make(chan string)

	r, w := io.Pipe()

	go func() {
		_, err := stdcopy.StdCopy(w, w, reader)
		if err != nil {
			t.logger.Errorf("StdCopy error: %v", err)
		}
		err = reader.Close()
		if err != nil {
			t.logger.Errorf("Failed to close reader: %v", err)
		}
		err = w.Close()
		if err != nil {
			t.logger.Errorf("Failed to close pipe writer: %v", err)
		}
	}()

	go func() {
		bufReader := bufio.NewReader(r)

		for {
			line, _, err := bufReader.ReadLine()
			if err != nil {
				break
			}
			ch <- string(line)
		}

		err = reader.Close()
		if err != nil {
			ch <- "Error: " + err.Error()
		}

		close(ch)
	}()

	return ch, nil
}

func (t *SingleContainerService) GetLogs(since string, tail string) (<-chan string, error) {
	return t.getLogs(since, tail, false)
}

func (t *SingleContainerService) FollowLogs(since string, tail string) (<-chan string, error) {
	return t.getLogs(since, tail, true)
}

func (t *SingleContainerService) Getenv(key string) (string, error) {
	c, err := t.getContainer()
	if err != nil {
		return "", err
	}
	prefix := key + "="
	for _, env := range c.Config.Env {
		if strings.HasPrefix(env, prefix) {
			value := strings.Replace(env, prefix, "", 1)
			return value, nil
		}
	}
	return "", errors.New("no such key: " + key)
}

func (t *SingleContainerService) Exec1(command []string) (string, error) {
	ctx := context.Background()
	createResp, err := t.dockerClient.ContainerExecCreate(ctx, t.containerName, types.ExecConfig{
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
	attachResp, err := t.dockerClient.ContainerExecAttach(ctx, execId, types.ExecConfig{
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

	inspectResp, err := t.dockerClient.ContainerExecInspect(ctx, execId)
	if err != nil {
		return "", err
	}

	exitCode := inspectResp.ExitCode

	if exitCode != 0 {
		return output.String(), errors.New("non-zero exit code")
	}

	return output.String(), nil
}

// Exec1 is a shortcut function
func (t *SingleContainerService) ExecInteractive(command []string) (string, io.Reader, io.Writer, error) {
	ctx := context.Background()
	createResp, err := t.dockerClient.ContainerExecCreate(ctx, t.containerName, types.ExecConfig{
		Cmd:          command,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", nil, nil, err
	}

	execId := createResp.ID

	t.logger.Infof("Created exec: %v", execId)

	// ContainerExecAttach = ContainerExecStart
	attachResp, err := t.dockerClient.ContainerExecAttach(ctx, execId, types.ExecConfig{})
	if err != nil {
		return execId, nil, nil, err
	}

	t.logger.Infof("Attached %v", attachResp)

	r, w := io.Pipe()

	go func() {
		_, err = stdcopy.StdCopy(w, w, attachResp.Reader)
		if err != nil {
			t.logger.Errorf("StdCopy failed: %v", err)
		}
		attachResp.Close()
	}()

	return execId, r, attachResp.Conn, nil
}

func (t *SingleContainerService) initContainer() {
	c, err := t.getContainer()
	if err != nil {
		t.logger.Debugf("Failed to get container %s while initializing", t.containerName)
	}
	t.setContainer(c)
}

func (t *SingleContainerService) setContainer(c *types.ContainerJSON) {
	t.cond.L.Lock()
	t.container = c
	if c != nil {
		t.cond.Broadcast()
	}
	t.cond.L.Unlock()
}

func (t *SingleContainerService) WaitContainer() *types.ContainerJSON {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	for t.container == nil {
		t.cond.Wait()
	}
	return t.container
}

func (t *SingleContainerService) OnEvent(type_ string) {
	var err error
	var c *types.ContainerJSON
	switch type_ {
	case "create":
		t.logger.Debugf("[Event] %s: Container %s created", t.name, t.containerName)
		c, err = t.getContainer()
		if err != nil {
			t.logger.Error("Failed to get container while CREATE event received: %s", err)
		}
		t.setContainer(c)
	case "start":
		t.logger.Debugf("[Event] %s: Container %s started", t.name, t.containerName)
		c, err = t.getContainer()
		if err != nil {
			t.logger.Error("Failed to get container while START event received: %s", err)
		}
		t.setContainer(c)
	case "die":
		t.logger.Debugf("[Event] %s: Container %s died", t.name, t.containerName)
		c, err = t.getContainer()
		if err != nil {
			t.logger.Error("Failed to get container while DIE event received: %s", err)
		}
		t.setContainer(c)
	case "destroy":
		t.logger.Debugf("[Event] %s: Container %s destroyed", t.name, t.containerName)
		t.setContainer(nil)
	}
}
