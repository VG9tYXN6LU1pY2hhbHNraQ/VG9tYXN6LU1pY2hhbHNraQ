package worker

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"app/storage"
)

type fetcher struct {
	fetchTimeout time.Duration
	manager      *Manager
	storage      storage.Storage
}

type Fetcher interface {
	Start(job storage.Job)
	Stop(id int)
}

func NewFetcher(st storage.Storage) Fetcher {
	return &fetcher{
		fetchTimeout: 5 * time.Second,
		manager:      NewManager(),
		storage:      st,
	}
}

func (f *fetcher) Start(job storage.Job) {
	interval := time.Duration(job.Interval * float64(time.Second))
	f.manager.Start(job.Id, interval, f.fetchAndSave(job.Id, job.Url))
}

func (f *fetcher) Stop(id int) {
	f.manager.Stop(id)
}

func (f *fetcher) fetchAndSave(id int, url string) func(ctx context.Context) {
	return func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, f.fetchTimeout)
		defer cancel()

		start := time.Now()
		response, err := fetchUrlContent(ctx, url)
		if err != nil {
			response = nil
		}
		duration := float64(time.Since(start).Milliseconds()) / 1000.0
		createdAt := unixWithFraction(time.Now())

		f.storage.AppendJobHistory(id, storage.HistoryEntry{
			Response:  response,
			Duration:  duration,
			CreatedAt: createdAt,
		})
	}
}

func unixWithFraction(t time.Time) float64 {
	return float64(t.UnixNano()) / 10e8
}

func fetchUrlContent(ctx context.Context, url string) (*string, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	buffer, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	str := string(buffer)
	return &str, nil
}
