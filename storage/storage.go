package storage

import (
	"sort"
	"sync"
)

type Storage interface {
	CreateRecord(record Record) Record
	GetRecords() []Record
}

func New() Storage {
	return &storage{
		mutex:   &sync.RWMutex{},
		records: map[int]Record{},
	}
}

type storage struct {
	lastId  int
	mutex   *sync.RWMutex
	records map[int]Record
}

func (s *storage) CreateRecord(record Record) Record {
	s.mutex.Lock()
	s.lastId++
	record.Id = s.lastId
	s.records[record.Id] = record
	s.mutex.Unlock()
	return record
}

func (s *storage) GetRecords() []Record {
	records := []Record{}
	s.mutex.RLock()
	for _, record := range s.records {
		records = append(records, record)
	}
	s.mutex.RUnlock()

	// sort records to make testing easier
	// in production, some real database would be used anyway so performance of this solution is not that big concern
	sort.Slice(records, func(i, j int) bool {
		return records[i].Id < records[j].Id
	})
	return records
}
