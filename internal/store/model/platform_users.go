package model

import (
	"time"
)


// PlatformUser mapped from table <platform_users>
type PlatformUser struct {
	ID         int64     `json:"id"`
	PlatformID int64     `json:"platform_id"`
	UserID     int64     `json:"user_id"`
	Flag       string    `json:"flag"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	AvatarURL  string    `json:"avatar_url"`
	IsBot      bool      `json:"is_bot"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
