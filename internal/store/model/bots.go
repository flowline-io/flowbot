package model

import (
	"time"
)

// Bot mapped from table <bots>
type Bot struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	State     BotState  `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
