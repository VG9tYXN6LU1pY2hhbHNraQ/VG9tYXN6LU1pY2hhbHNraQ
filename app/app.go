package app

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"

	"app/storage"
)

type Instance struct {
	handler http.Handler
	storage storage.Storage
}

func NewInstance() *Instance {
	router := mux.NewRouter()
	instance := &Instance{handler: router, storage: storage.New()}
	router.HandleFunc("/", instance.index)
	router.HandleFunc("/api/fetcher", instance.getRecords)
	return instance
}

func (i *Instance) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	i.handler.ServeHTTP(writer, request)
}

func (i *Instance) index(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, "Hello world!")
}

func (i *Instance) getRecords(w http.ResponseWriter, r *http.Request) {
	records := i.storage.GetRecords()
	response, _ := json.Marshal(records)
	_, _ = fmt.Fprint(w, string(response))
}
