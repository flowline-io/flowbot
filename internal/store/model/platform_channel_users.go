package model

import (
	"time"
)


// PlatformChannelUser mapped from table <platform_channel_users>
type PlatformChannelUser struct {
	ID          int64     `json:"id"`
	PlatformID  int64     `json:"platform_id"`
	ChannelFlag string    `json:"channel_flag"`
	UserFlag    string    `json:"user_flag"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
