package core

import (
	"github.com/gin-gonic/gin"
	"io"
)

type Listener interface {
	OnEvent(type_ string)
}

type Service interface {
	io.Closer
	Listener

	GetName() string
	GetStatus() (string, error)
	ConfigureRouter(r *gin.RouterGroup)
	GetLogs(since string, tail string) (<-chan string, error)

	IsDisabled() bool
	SetDisabled(value bool)
	GetContainerId() string
	GetMode() string
	SetMode(value string)
}
