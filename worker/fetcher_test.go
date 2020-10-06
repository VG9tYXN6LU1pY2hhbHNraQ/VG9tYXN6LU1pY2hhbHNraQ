package worker_test

import (
	"app/storage"
	"app/worker"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetcher(t *testing.T) {
	counter := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(fmt.Sprintf("foobar-%d", counter)))
		counter++
	}))
	defer server.Close()

	st := storage.New()
	fetcher := worker.NewFetcher(st)

	record := st.CreateRecord(storage.Record{
		Url:      server.URL + "/foo",
		Interval: 0.15,
	})
	fetcher.Start(record)
	time.Sleep(700 * time.Millisecond)
	fetcher.Stop(record.Id)
	time.Sleep(100 * time.Millisecond)

	history, _ := st.GetRecordHistory(record.Id)
	for i := 0; i < len(history); i++ {
		entry := history[i]
		expectedResponse := fmt.Sprintf("foobar-%d", i)
		if entry.Response != nil && *entry.Response != expectedResponse {
			t.Errorf("Expected response to be '%s'. Got '%s'", expectedResponse, *entry.Response)
		}
		if !(entry.Duration > 0) {
			t.Errorf("Expected duration to be > 0. Got '%f'", entry.Duration)
		}
		if !(entry.CreatedAt > 0) {
			t.Errorf("Expected creation date to be > 0. Got '%f'", entry.CreatedAt)
		}
	}
}
