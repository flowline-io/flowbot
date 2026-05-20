package model

import (
	"time"
)

// Config mapped from table <configs>
type Config struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Key       string    `json:"key"`
	Value     JSON      `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
