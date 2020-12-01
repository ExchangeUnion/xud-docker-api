package core

import (
	"bufio"
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
)

type Container struct {
	c      *types.ContainerJSON
	client *docker.Client
	logger *logrus.Logger
}

func (t *Container) Unwrap() *types.ContainerJSON {
	return t.c
}

func (t *Container) GetStatus() string {
	return t.c.State.Status
}

func (t *Container) Exec(command []string) (string, error) {
	ctx := context.Background()
	createResp, err := t.client.ContainerExecCreate(ctx, t.c.ID, types.ExecConfig{
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
	attachResp, err := t.client.ContainerExecAttach(ctx, execId, types.ExecConfig{
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

	inspectResp, err := t.client.ContainerExecInspect(ctx, execId)
	if err != nil {
		return "", err
	}

	exitCode := inspectResp.ExitCode

	if exitCode != 0 {
		return output.String(), errors.New("non-zero exit code")
	}

	return output.String(), nil
}

func (t *Container) ExecInteractive(command []string) (string, io.Reader, io.Writer, error) {

	ctx := context.Background()
	createResp, err := t.client.ContainerExecCreate(ctx, t.c.ID, types.ExecConfig{
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
	attachResp, err := t.client.ContainerExecAttach(ctx, execId, types.ExecConfig{})
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

func (t *Container) GetLogs(since string, tail string, follow bool) (<-chan string, error) {
	reader, err := t.client.ContainerLogs(context.Background(), t.c.ID, types.ContainerLogsOptions{
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
		//t.logger.Infof("StdCopy %d bytes", n)
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

func (t *Container) Getenv(key string) string {
	prefix := key + "="
	for _, env := range t.c.Config.Env {
		if strings.HasPrefix(env, prefix) {
			value := strings.Replace(env, prefix, "", 1)
			return value
		}
	}
	return ""
}
