package service

import (
	"encoding/json"
	"fmt"
	"github.com/ExchangeUnion/xud-docker-api-poc/utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
)

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

func (t *AbstractService) SetServiceManager(serviceManager ServiceManager) {
	t.serviceManager = serviceManager
}

func (t *AbstractService) GetServiceManager() ServiceManager {
	return t.serviceManager
}

func (t *AbstractService) ConfigureRouter(r *mux.Router) {
	path := fmt.Sprintf("/api/v1/status/%s", t.GetName())
	r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		log.Printf("t type is %v", reflect.TypeOf(t))
		status, err:= t.GetStatus()
		if err != nil {
			utils.JsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(status)
		if err != nil {
			utils.JsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")
}

func (t *AbstractService) Close() {
}
