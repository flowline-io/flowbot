package model

import (
	"time"
)


// Connection mapped from table <connections>
type Connection struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    JSON      `json:"config"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
