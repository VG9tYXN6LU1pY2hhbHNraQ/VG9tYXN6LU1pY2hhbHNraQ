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

var defaultTestJobs = []storage.Job{{
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

func TestGetJobs(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("GET", "/api/fetcher", "")

	expected := `[` +
		`{"id":1,"url":"https://httpbin.org/range/15","interval":60},` +
		`{"id":2,"url":"https://httpbin.org/delay/10","interval":120}` +
		`]`
	assertResponse(t, response, http.StatusOK, expected)
}

func TestCreateJob(t *testing.T) {
	job := storage.Job{
		Id:       3,
		Url:      "http://example.com/",
		Interval: 42,
	}

	i := newTestAppInstance()
	assertJobs(t, i.Storage.GetJobs(), defaultTestJobs)

	body := fmt.Sprintf(`{"url":"%s","interval":%f}`, job.Url, job.Interval)
	response := i.doRequest("POST", "/api/fetcher", body)
	expected := fmt.Sprintf(`{"id":%d}`, job.Id)
	assertResponse(t, response, http.StatusCreated, expected)

	assertJobs(t, i.Storage.GetJobs(), append(defaultTestJobs, job))
}

func TestCreateJobWithInvalidPayload(t *testing.T) {
	i := newTestAppInstance()
	i.RequestMaxBytes = 5

	response := i.doRequest("POST", "/api/fetcher", "foo")
	assertResponse(t, response, http.StatusBadRequest, "")

	response = i.doRequest("POST", "/api/fetcher", `{"foobar":"foobar"}`)
	assertResponse(t, response, http.StatusRequestEntityTooLarge, "")

	assertJobs(t, i.Storage.GetJobs(), append(defaultTestJobs))
}

func TestDeleteJob(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("DELETE", "/api/fetcher/1", "")

	assertResponse(t, response, http.StatusNoContent, "")
	assertJobs(t, i.Storage.GetJobs(), defaultTestJobs[1:])

	response = i.doRequest("DELETE", "/api/fetcher/1", "")
	assertResponse(t, response, http.StatusNotFound, "")

	response = i.doRequest("DELETE", "/api/fetcher/foobar", "")
	assertResponse(t, response, http.StatusNotFound, "")
}

func TestManageJobsConcurrently(t *testing.T) {
	// meant to be run with -race flag

	jobs := []storage.Job{{
		Url:      "http://example.com/A",
		Interval: 42,
	}, {
		Url:      "http://example.com/B",
		Interval: 42,
	}}

	deletedJobId := defaultTestJobs[0].Id

	i := newTestAppInstance()
	assertJobs(t, i.Storage.GetJobs(), defaultTestJobs)

	wg := sync.WaitGroup{}
	wg.Add(len(jobs) + 2)
	for _, job := range jobs {
		go func(job storage.Job) {
			body := fmt.Sprintf(`{"url":"%s","interval":%f}`, job.Url, job.Interval)
			_ = i.doRequest("POST", "/api/fetcher", body)
			wg.Done()
		}(job)
	}
	go func() {
		// check whether existing jobs can be accessed while a new one is added
		// there is no assertion because there are multiple possibilities of valid responses
		// however relying on go race detector is sufficient
		_ = i.doRequest("GET", "/api/fetcher", "")
		wg.Done()
	}()
	go func() {
		_ = i.doRequest("DELETE", fmt.Sprintf("/api/fetcher/%d", deletedJobId), "")
		wg.Done()
	}()
	wg.Wait()

	savedJobs := i.Storage.GetJobs()
	for _, job := range jobs {
		assertJobsContainUrl(t, savedJobs, job.Url)
	}
	assertJobsDoNotContainId(t, savedJobs, deletedJobId)
}

func TestGetJobHistory(t *testing.T) {
	i := newTestAppInstance()

	response := i.doRequest("GET", "/api/fetcher/1/history", "")
	expected := `[` +
		`{"response":"abcdefghijklmno","duration":0.571,"created_at":1559034638.31525},` +
		`{"response":null,"duration":5,"created_at":1559034938.623}` +
		`]`
	assertResponse(t, response, http.StatusOK, expected)

	i.Storage.AppendJobHistory(1, storage.HistoryEntry{
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
	for _, job := range defaultTestJobs {
		instance.Storage.CreateJob(job)
	}
	instance.Fetcher = dummyFetcher{}
	return &testAppInstance{Instance: instance}
}

type dummyFetcher struct{}

func (dummyFetcher) Start(job storage.Job) {}
func (dummyFetcher) Stop(id int)           {}

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

func assertJobs(t *testing.T, actual, expected []storage.Job) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected jobs to be '%#v'. Got '%#v'", expected, actual)
	}
}

func assertJobsContainUrl(t *testing.T, jobs []storage.Job, url string) {
	found := false
	for _, job := range jobs {
		if job.Url == url {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected jobs to contain url '%s'. Got '%#v'", url, jobs)
	}
}

func assertJobsDoNotContainId(t *testing.T, jobs []storage.Job, id int) {
	found := false
	for _, job := range jobs {
		if job.Id == id {
			found = true
			break
		}
	}

	if found {
		t.Errorf("Expected jobs to not contain id '%d'. Got '%#v'", id, jobs)
	}
}
