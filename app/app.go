package app

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

type Instance struct {
	handler http.Handler
}

func NewInstance() *Instance {
	router := mux.NewRouter()
	instance := &Instance{handler: router}
	router.HandleFunc("/", instance.index)
	return instance
}

func (i *Instance) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	i.handler.ServeHTTP(writer, request)
}

func (i *Instance) index(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, "Hello world!")
}
