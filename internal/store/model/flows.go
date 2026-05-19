package model

import (
	"time"
)


// Flow mapped from table <flows>
type Flow struct {
	ID          int64     `json:"id"`
	UID         string    `json:"uid"`
	Topic       string    `json:"topic"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	State       FlowState `json:"state"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
