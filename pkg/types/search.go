package types

type Document struct {
	Id          string `json:"id"`
	SourceId    string `json:"source_id"`
	Source      string `json:"source"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	CreatedAt   int64  `json:"created_at"`
}
