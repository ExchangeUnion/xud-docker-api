package core

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AbstractService struct {
	name     string
	services map[string]Service
	logger   *logrus.Entry

	disabled bool
	mode     string
}

func NewAbstractService(name string, services map[string]Service) *AbstractService {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: false,
	})

	logger := log.WithField("service", name)

	return &AbstractService{
		name:     name,
		services: services,
		logger:   logger,
		disabled: false,
		mode:     "",
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

func (t *AbstractService) GetLogger() *logrus.Entry {
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

func (t *AbstractService) GetMode() string {
	return t.mode
}

func (t *AbstractService) SetMode(value string) {
	t.mode = value
}
