package model

import (
	"time"
)

// PlatformBot mapped from table <platform_bots>
type PlatformBot struct {
	ID         int64     `json:"id"`
	PlatformID int64     `json:"platform_id"`
	BotID      int64     `json:"bot_id"`
	Flag       string    `json:"flag"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
