package model

import (
	"time"
)


// PlatformChannel mapped from table <platform_channels>
type PlatformChannel struct {
	ID         int64     `json:"id"`
	PlatformID int64     `json:"platform_id"`
	ChannelID  int64     `json:"channel_id"`
	Flag       string    `json:"flag"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
