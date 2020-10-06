package app

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"app/storage"
	"app/worker"
)

type Instance struct {
	handler         http.Handler
	Fetcher         worker.Fetcher
	Storage         storage.Storage
	RequestMaxBytes int64
}

func NewInstance() *Instance {
	router := mux.NewRouter()
	st := storage.New()
	instance := &Instance{
		handler:         router,
		Fetcher:         worker.NewFetcher(st),
		Storage:         st,
		RequestMaxBytes: 1024 * 1024,
	}
	router.HandleFunc("/", instance.index)
	router.HandleFunc("/api/fetcher", instance.getRecords).Methods("GET")
	router.HandleFunc("/api/fetcher", instance.createRecord).Methods("POST")
	router.HandleFunc("/api/fetcher/{id}", instance.deleteRecord).Methods("DELETE")
	router.HandleFunc("/api/fetcher/{id}/history", instance.getRecordHistory).Methods("GET")
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
		Url      string  `json:"url"`
		Interval float64 `json:"interval"`
	}{}
	err := jsonDecode(http.MaxBytesReader(w, r.Body, i.RequestMaxBytes), &request)
	if err != nil {
		log.Printf("create record: %s", err)
		if isRequestBodyTooLarge(err) {
			writeResponse(w, http.StatusRequestEntityTooLarge, nil)
		} else {
			writeResponse(w, http.StatusBadRequest, nil)
		}
		return
	}

	record := i.Storage.CreateRecord(storage.Record{
		Url:      request.Url,
		Interval: request.Interval,
	})
	i.Fetcher.Start(record)
	writeResponse(w, http.StatusCreated, struct {
		Id int `json:"id"`
	}{
		Id: record.Id,
	})
}

func (i *Instance) deleteRecord(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromUrl(r)
	if err != nil {
		writeResponse(w, http.StatusNotFound, nil)
		return
	}

	i.Fetcher.Stop(id)
	existed := i.Storage.DeleteRecord(id)
	if existed {
		writeResponse(w, http.StatusNoContent, nil)
	} else {
		writeResponse(w, http.StatusNotFound, nil)
	}
}

func (i *Instance) getRecordHistory(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromUrl(r)
	if err != nil {
		writeResponse(w, http.StatusNotFound, nil)
		return
	}

	history, exists := i.Storage.GetRecordHistory(id)
	if exists {
		writeResponse(w, http.StatusOK, history)
	} else {
		writeResponse(w, http.StatusNotFound, nil)
	}
}

func jsonDecode(reader io.ReadCloser, value interface{}) error {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(value)

	if err != nil {
		return fmt.Errorf(`json decode: %w`, err)
	}
	return nil
}

func isRequestBodyTooLarge(err error) bool {
	// sadly there is no sentinel error to check against, so comparing strings is the only way
	return strings.Contains(err.Error(), "http: request body too large")
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

func getIdFromUrl(r *http.Request) (int, error) {
	idString, _ := mux.Vars(r)["id"]
	return strconv.Atoi(idString)
}
