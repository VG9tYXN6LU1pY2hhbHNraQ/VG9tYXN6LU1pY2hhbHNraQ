package app

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"

	"app/storage"
)

type Instance struct {
	handler http.Handler
	Storage storage.Storage
}

func NewInstance() *Instance {
	router := mux.NewRouter()
	instance := &Instance{handler: router, Storage: storage.New()}
	router.HandleFunc("/", instance.index)
	router.HandleFunc("/api/fetcher", instance.getRecords).Methods("GET")
	router.HandleFunc("/api/fetcher", instance.createRecord).Methods("POST")
	router.HandleFunc("/api/fetcher/{id}", instance.deleteRecord).Methods("DELETE")
	return instance
}

func (i *Instance) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	i.handler.ServeHTTP(writer, request)
}

func (i *Instance) index(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, http.StatusOK, "Hello world!")
}

func (i *Instance) getRecords(w http.ResponseWriter, r *http.Request) {
	records := i.Storage.GetRecords()
	writeResponse(w, http.StatusOK, records)
}

func (i *Instance) createRecord(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Url      string `json:"url"`
		Interval int    `json:"interval"`
	}{}
	jsonDecode(r.Body, &request)

	record := i.Storage.CreateRecord(storage.Record{
		Url:      request.Url,
		Interval: request.Interval,
	})
	writeResponse(w, http.StatusCreated, struct {
		Id int `json:"id"`
	}{
		Id: record.Id,
	})
}

func (i *Instance) deleteRecord(w http.ResponseWriter, r *http.Request) {
	idString, _ := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idString)
	if err != nil {
		writeResponse(w, http.StatusNotFound, nil)
		return
	}

	existed := i.Storage.DeleteRecord(id)
	if existed {
		writeResponse(w, http.StatusNoContent, nil)
	} else {
		writeResponse(w, http.StatusNotFound, nil)
	}
}

func jsonDecode(reader io.ReadCloser, value interface{}) {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(value)

	if err != nil {
		err = fmt.Errorf(`json decode: %w`, err)
		log.Fatal(err)
	}
}

func writeResponse(w http.ResponseWriter, statusCode int, v interface{}) {
	w.WriteHeader(statusCode)
	if v == nil {
		return
	}

	response, err := json.Marshal(v)
	if err != nil {
		err = fmt.Errorf(`write response: json marshal: %w`, err)
		log.Fatalf("%s", err)
	}
	_, err = fmt.Fprint(w, string(response))
	if err != nil {
		err = fmt.Errorf(`write response: print: %w`, err)
		log.Fatalf("%s", err)
	}
}
