package storage

import "sort"

type Storage interface {
	CreateRecord(record Record) Record
	GetRecords() []Record
}

func New() Storage {
	return &storage{
		records: map[int]Record{},
	}
}

type storage struct {
	lastId  int
	records map[int]Record
}

func (s *storage) CreateRecord(record Record) Record {
	// TODO add mutex along with test for -race flag
	s.lastId++
	record.Id = s.lastId
	s.records[record.Id] = record
	return record
}

func (s *storage) GetRecords() []Record {
	// TODO add mutex along with test for -race flag
	records := []Record{}
	for _, record := range s.records {
		records = append(records, record)
	}

	// sort records to make testing easier
	// in production, some real database would be used anyway so performance of this solution is not that big concern
	sort.Slice(records, func(i, j int) bool {
		return records[i].Id < records[j].Id
	})
	return records
}
