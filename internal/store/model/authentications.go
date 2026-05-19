package model

import (
	"time"
)


// Authentication mapped from table <authentications>
type Authentication struct {
	ID           int64      `json:"id"`
	UID          string     `json:"uid"`
	Topic        string     `json:"topic"`
	ConnectionID *int64     `json:"connection_id"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Credentials  JSON       `json:"credentials"`
	ExpiresAt    *time.Time `json:"expires_at"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
