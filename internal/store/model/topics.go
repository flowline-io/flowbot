package model

import (
	"time"
)


// Topic mapped from table <topics>
type Topic struct {
	ID        int64      `json:"id"`
	Flag      string     `json:"flag"`
	Platform  string     `json:"platform"`
	Owner     int64      `json:"owner"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Tags      string     `json:"tags"`
	State     TopicState `json:"state"`
	TouchedAt time.Time  `json:"touched_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
