package model

import (
	"time"
)


// Page mapped from table <pages>
type Page struct {
	ID        int64     `json:"id"`
	PageID    string    `json:"page_id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Type      PageType  `json:"type"`
	Schema    JSON      `json:"schema"`
	State     PageState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
