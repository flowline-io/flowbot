package model

import (
	"time"
)

// User mapped from table <users>
type User struct {
	ID        int64     `json:"id"`
	Flag      string    `json:"flag"`
	Name      string    `json:"name"`
	Tags      string    `json:"tags"`
	State     UserState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
