package model

import (
	"time"
)


// Message mapped from table <messages>
type Message struct {
	ID            int64          `json:"id"`
	Flag          string         `json:"flag"`
	PlatformID    int64          `json:"platform_id"`
	PlatformMsgID string         `json:"platform_msg_id"`
	Topic         string         `json:"topic"`
	Role          string         `json:"role"`
	Session       string         `json:"session"`
	Content       JSON           `json:"content"`
	State         MessageState   `json:"state"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     *time.Time     `json:"deleted_at"`
}
