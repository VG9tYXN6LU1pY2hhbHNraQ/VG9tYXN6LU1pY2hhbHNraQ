package app_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"app/app"
	"app/storage"
)

var defaultTestRecords = []storage.Record{{
	Id:       1,
	Url:      "https://httpbin.org/range/15",
	Interval: 60,
}, {
	Id:       2,
	Url:      "https://httpbin.org/delay/10",
	Interval: 120,
}}

func TestIndex(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("GET", "/", "")
	assertResponse(t, response, http.StatusOK, `"Hello world!"`)
}

func TestGetRecords(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("GET", "/api/fetcher", "")

	expected := `[` +
		`{"id":1,"url":"https://httpbin.org/range/15","interval":60},` +
		`{"id":2,"url":"https://httpbin.org/delay/10","interval":120}` +
		`]`
	assertResponse(t, response, http.StatusOK, expected)
}

func TestCreateRecord(t *testing.T) {
	record := storage.Record{
		Id:       3,
		Url:      "http://example.com/",
		Interval: 42,
	}

	i := newTestAppInstance()
	assertRecords(t, i.Storage.GetRecords(), defaultTestRecords)

	body := fmt.Sprintf(`{"url":"%s","interval":%d}`, record.Url, record.Interval)
	response := i.doRequest("POST", "/api/fetcher", body)
	expected := fmt.Sprintf(`{"id":%d}`, record.Id)
	assertResponse(t, response, http.StatusCreated, expected)

	assertRecords(t, i.Storage.GetRecords(), append(defaultTestRecords, record))
}

func newTestAppInstance() *testAppInstance {
	instance := app.NewInstance()
	for _, record := range defaultTestRecords {
		instance.Storage.CreateRecord(record)
	}
	return &testAppInstance{Instance: instance}
}

type testAppInstance struct {
	*app.Instance
}

func (i *testAppInstance) doRequest(method string, url string, body string) *httptest.ResponseRecorder {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	response := httptest.NewRecorder()
	i.ServeHTTP(response, request)
	return response
}

func assertResponse(t *testing.T, response *httptest.ResponseRecorder, code int, body string) {
	if response.Code != code {
		t.Errorf("Expected response code to be '%d'. Got '%d'", code, response.Code)
	}

	if response.Body.String() != body {
		t.Errorf("Expected response body to be '%s'. Got '%s'", body, response.Body.String())
	}
}

func assertRecords(t *testing.T, actual, expected []storage.Record) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected records to be '%#v'. Got '%#v'", expected, actual)
	}
}
