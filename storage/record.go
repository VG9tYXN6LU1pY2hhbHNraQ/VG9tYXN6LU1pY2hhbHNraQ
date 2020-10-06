package storage

type Record struct {
	Id       int            `json:"id"`
	Url      string         `json:"url"`
	Interval int            `json:"interval"`
	History  []HistoryEntry `json:"-"`
}

type HistoryEntry struct {
	Response  *string `json:"response"`
	Duration  float64 `json:"duration"`
	CreatedAt float64 `json:"created_at"`
}

func OptionalString(s string) *string {
	return &s
}
