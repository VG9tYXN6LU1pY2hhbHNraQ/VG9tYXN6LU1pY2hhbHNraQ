package storage

type Storage interface {
	GetRecords() []Record
}

func New() Storage {
	return &storage{}
}

type storage struct{}

func (s *storage) GetRecords() []Record {
	return []Record{{
		Id:       1,
		Url:      "https://httpbin.org/range/15",
		Interval: 60,
	}, {
		Id:       2,
		Url:      "https://httpbin.org/delay/10",
		Interval: 120,
	}}
}
