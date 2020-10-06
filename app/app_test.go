package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/app"
)

func TestIndex(t *testing.T) {
	i := newTestAppInstance()
	response := i.doRequest("GET", "/", "")
	assertResponse(t, response, http.StatusOK, "Hello world!")
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

func newTestAppInstance() *testAppInstance {
	instance := app.NewInstance()
	return &testAppInstance{Instance: instance}
}

type testAppInstance struct {
	*app.Instance
}

func (i *testAppInstance) doRequest(method string, url string, body string) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(method, url, strings.NewReader(body))
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
