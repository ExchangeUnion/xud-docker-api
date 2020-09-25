package service

import "github.com/gorilla/mux"

type AbstractService struct {
	name           string
	serviceManager ServiceManager
}

func NewAbstractService(name string) *AbstractService {
	return &AbstractService{
		name: name,
	}
}

func (t *AbstractService) GetName() string {
	return t.name
}

func (t *AbstractService) GetStatus() (string, error) {
	return "Unknown", nil
}

func (t *AbstractService) ConfigureRouter(r *mux.Router) {
}

func (t *AbstractService) Close() {
}

func (t *AbstractService) SetServiceManager(serviceManager ServiceManager) {
	t.serviceManager = serviceManager
}

func (t *AbstractService) GetServiceManager() ServiceManager {
	return t.serviceManager
}
