package storage

type Record struct {
	Id       int    `json:"id"`
	Url      string `json:"url"`
	Interval int    `json:"interval"`
}
