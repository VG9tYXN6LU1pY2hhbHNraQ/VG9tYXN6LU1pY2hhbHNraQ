package storage

import (
	"sort"
	"sync"
)

type Storage interface {
	CreateJob(job Job) Job
	GetJobs() []Job
	DeleteJob(id int) bool
	AppendJobHistory(id int, entry HistoryEntry)
	GetJobHistory(id int) ([]HistoryEntry, bool)
}

func New() Storage {
	return &storage{
		mutex: &sync.RWMutex{},
		jobs:  map[int]Job{},
	}
}

type storage struct {
	lastId int
	mutex  *sync.RWMutex
	jobs   map[int]Job
}

func (s *storage) CreateJob(job Job) Job {
	s.mutex.Lock()
	s.lastId++
	job.Id = s.lastId
	s.jobs[job.Id] = job
	s.mutex.Unlock()
	return job
}

func (s *storage) GetJobs() []Job {
	jobs := []Job{}
	s.mutex.RLock()
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.mutex.RUnlock()

	// sort jobs to make testing easier
	// in production, some real database would be used anyway so performance of this solution is not that big concern
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Id < jobs[j].Id
	})
	return jobs
}

func (s *storage) GetJobHistory(id int) ([]HistoryEntry, bool) {
	s.mutex.RLock()
	job, exists := s.jobs[id]
	s.mutex.RUnlock()
	history := job.History
	if job.History == nil {
		history = []HistoryEntry{}
	}
	return history, exists
}

func (s *storage) DeleteJob(id int) bool {
	s.mutex.Lock()
	_, exists := s.jobs[id]
	if exists {
		delete(s.jobs, id)
	}
	s.mutex.Unlock()
	return exists
}

func (s *storage) AppendJobHistory(id int, entry HistoryEntry) {
	s.mutex.Lock()
	job, exists := s.jobs[id]
	if exists {
		job.History = append(job.History, entry)
		s.jobs[id] = job
	}
	s.mutex.Unlock()
}
