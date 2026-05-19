package model

import (
	"time"
)

// Channel mapped from table <channels>
type Channel struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Flag      string       `json:"flag"`
	State     ChannelState `json:"state"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
