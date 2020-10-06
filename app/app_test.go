package app_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"app/app"
	"app/storage"
)

var defaultTestRecords = []storage.Record{{
	Id:       1,
	Url:      "https://httpbin.org/range/15",
	Interval: 60,
	History: []storage.HistoryEntry{{
		Response:  storage.OptionalString("abcdefghijklmno"),
		Duration:  0.571,
		CreatedAt: 1559034638.31525,
	}, {
		Response:  nil,
		Duration:  5,
		CreatedAt: 1559034938.623,
	}},
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

	body := fmt.Sprintf(`{"url":"%s","interval":%f}`, record.Url, record.Interval)
	response := i.doRequest("POST", "/api/fetcher", body)
	expected := fmt.Sprintf(`{"id":%d}`, record.Id)
	assertResponse(t, response, http.StatusCreated, expected)

	assertRecords(t, i.Storage.GetRecords(), append(defaultTestRecords, record))
}

func TestCreateRecordWithInvalidPayload(t *testing.T) {
	i := newTestAppInstance()
	i.RequestMaxBytes = 5

	response := i.doRequest("POST", "/api/fetcher", "foo")
	assertResponse(t, response, http.StatusBadRequest, "")

	response = i.doRequest("POST", "/api/fetcher", `{"foobar":"foobar"}`)
	assertResponse(t, response, http.StatusRequestEntityTooLarge, "")

	assertRecords(t, i.Storage.GetRecords(), append(defaultTestRecords))
}

func TestDeleteRecord(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("DELETE", "/api/fetcher/1", "")

	assertResponse(t, response, http.StatusNoContent, "")
	assertRecords(t, i.Storage.GetRecords(), defaultTestRecords[1:])

	response = i.doRequest("DELETE", "/api/fetcher/1", "")
	assertResponse(t, response, http.StatusNotFound, "")

	response = i.doRequest("DELETE", "/api/fetcher/foobar", "")
	assertResponse(t, response, http.StatusNotFound, "")
}

func TestManageRecordsConcurrently(t *testing.T) {
	// meant to be run with -race flag

	records := []storage.Record{{
		Url:      "http://example.com/A",
		Interval: 42,
	}, {
		Url:      "http://example.com/B",
		Interval: 42,
	}}

	deletedRecordId := defaultTestRecords[0].Id

	i := newTestAppInstance()
	assertRecords(t, i.Storage.GetRecords(), defaultTestRecords)

	wg := sync.WaitGroup{}
	wg.Add(len(records) + 2)
	for _, record := range records {
		go func(record storage.Record) {
			body := fmt.Sprintf(`{"url":"%s","interval":%f}`, record.Url, record.Interval)
			_ = i.doRequest("POST", "/api/fetcher", body)
			wg.Done()
		}(record)
	}
	go func() {
		// check whether existing records can be accessed while a new one is added
		// there is no assertion because there are multiple possibilities of valid responses
		// however relying on go race detector is sufficient
		_ = i.doRequest("GET", "/api/fetcher", "")
		wg.Done()
	}()
	go func() {
		_ = i.doRequest("DELETE", fmt.Sprintf("/api/fetcher/%d", deletedRecordId), "")
		wg.Done()
	}()
	wg.Wait()

	savedRecords := i.Storage.GetRecords()
	for _, record := range records {
		assertRecordsContainUrl(t, savedRecords, record.Url)
	}
	assertRecordsDoNotContainId(t, savedRecords, deletedRecordId)
}

func TestGetRecordHistory(t *testing.T) {
	i := newTestAppInstance()

	response := i.doRequest("GET", "/api/fetcher/1/history", "")
	expected := `[` +
		`{"response":"abcdefghijklmno","duration":0.571,"created_at":1559034638.31525},` +
		`{"response":null,"duration":5,"created_at":1559034938.623}` +
		`]`
	assertResponse(t, response, http.StatusOK, expected)

	i.Storage.AppendRecordHistory(1, storage.HistoryEntry{
		Response:  storage.OptionalString("foobar"),
		Duration:  42,
		CreatedAt: 42,
	})

	response = i.doRequest("GET", "/api/fetcher/1/history", "")
	expected = `[` +
		`{"response":"abcdefghijklmno","duration":0.571,"created_at":1559034638.31525},` +
		`{"response":null,"duration":5,"created_at":1559034938.623},` +
		`{"response":"foobar","duration":42,"created_at":42}` +
		`]`
	assertResponse(t, response, http.StatusOK, expected)
}

func newTestAppInstance() *testAppInstance {
	instance := app.NewInstance()
	for _, record := range defaultTestRecords {
		instance.Storage.CreateRecord(record)
	}
	instance.Fetcher = dummyFetcher{}
	return &testAppInstance{Instance: instance}
}

type dummyFetcher struct{}

func (dummyFetcher) Start(record storage.Record) {}
func (dummyFetcher) Stop(id int)                 {}

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

func assertRecordsContainUrl(t *testing.T, records []storage.Record, url string) {
	found := false
	for _, record := range records {
		if record.Url == url {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected records to contain url '%s'. Got '%#v'", url, records)
	}
}

func assertRecordsDoNotContainId(t *testing.T, records []storage.Record, id int) {
	found := false
	for _, record := range records {
		if record.Id == id {
			found = true
			break
		}
	}

	if found {
		t.Errorf("Expected records to not contain id '%d'. Got '%#v'", id, records)
	}
}
