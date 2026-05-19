package model

import (
	"time"
)


// Url mapped from table <urls>
type Url struct {
	ID        int64     `json:"id"`
	Flag      string    `json:"flag"`
	URL       string    `json:"url"`
	State     UrlState  `json:"state"`
	ViewCount int32     `json:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
