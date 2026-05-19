package model

import (
	"time"
)


// OAuth mapped from table <oauth>
type OAuth struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Token     string    `json:"token"`
	Extra     JSON      `json:"extra"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
