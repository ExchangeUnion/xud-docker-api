package service

import (
	"github.com/gin-gonic/gin"
)

type Service interface {
	GetName() string
	GetStatus() (string, error)
	ConfigureRouter(r *gin.Engine)
	Close()
}

type ServiceManager interface {
	GetService(name string) (Service, error)
}
