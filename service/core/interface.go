package core

import (
	"context"
	"github.com/gin-gonic/gin"
	"io"
)

type DockerEventListener interface {
	OnEvent(type_ string)
}

type Service interface {
	io.Closer
	DockerEventListener

	ConfigureRouter(r *gin.RouterGroup)

	GetName() string
	GetStatus(ctx context.Context) string
	GetContainerId() string
	IsDisabled() bool
	SetDisabled(value bool)
	GetMode() string
	SetMode(value string)

	GetLogs(since string, tail string) ([]string, error)
	FollowLogs(since string, tail string) (<-chan string, func(), error)
}
