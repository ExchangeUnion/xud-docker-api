package service

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AbstractService struct {
	name           string
	serviceManager ServiceManager
	logger         *logrus.Logger
}

func NewAbstractService(name string) *AbstractService {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return &AbstractService{
		name:   name,
		logger: logger,
	}
}

func (t *AbstractService) GetName() string {
	return t.name
}

func (t *AbstractService) GetStatus() (string, error) {
	return "Unknown", nil
}

func (t *AbstractService) ConfigureRouter(r *gin.RouterGroup) {
}

func (t *AbstractService) Close() {
}

func (t *AbstractService) GetLogs(since string, tail string) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		close(ch)
	}()
	return ch, nil
}

func (t *AbstractService) SetServiceManager(serviceManager ServiceManager) {
	t.serviceManager = serviceManager
}

func (t *AbstractService) GetServiceManager() ServiceManager {
	return t.serviceManager
}

func (t *AbstractService) GetLogger() *logrus.Logger {
	return t.logger
}
