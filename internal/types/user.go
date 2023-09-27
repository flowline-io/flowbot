package types

type User struct {
	ID   int64  `json:"id"`
	Flag string `json:"flag"`
	Name string `json:"name"`
	Tags string `json:"tags"`
}
