package service

import (
	"github.com/gin-gonic/gin"
)

type Service interface {
	GetName() string
	GetStatus() (string, error)
	ConfigureRouter(r *gin.RouterGroup)
	Close()
	GetLogs(since string, tail string) (<-chan string, error)
}

type ServiceManager interface {
	GetService(name string) (Service, error)
}
