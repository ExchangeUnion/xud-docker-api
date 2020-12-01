package core

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AbstractService struct {
	name     string
	services map[string]Service
	logger   *logrus.Logger

	disabled bool
}

func NewAbstractService(name string, services map[string]Service) *AbstractService {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return &AbstractService{
		name:     name,
		services: services,
		logger:   logger,
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

func (t *AbstractService) GetLogger() *logrus.Logger {
	return t.logger
}

func (t *AbstractService) GetService(name string) Service {
	return t.services[name]
}

func (t *AbstractService) IsDisabled() bool {
	//key := fmt.Sprintf("XUD_DOCKER_SERVICE_%s_DISABLED", strings.ToUpper(t.GetName()))
	//value := os.Getenv(key)
	//if value == "true" {
	//	return true
	//}
	//return false
	return t.disabled
}

func (t *AbstractService) SetDisabled(value bool) {
	t.disabled = value
}
