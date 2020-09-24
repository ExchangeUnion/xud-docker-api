package service

import (
	"github.com/gorilla/mux"
)

type Service interface {
	GetName() string
	GetStatus() (string, error)
	ConfigureRouter(r *mux.Router)
	Close()
}

type ServiceManager interface {
	GetService(name string) (Service, error)
}
