package model

import (
	"time"
)

// Webhook mapped from table <webhook>
type Webhook struct {
	ID           int64        `json:"id"`
	UID          string       `json:"uid"`
	Topic        string       `json:"topic"`
	Flag         string       `json:"flag"`
	Secret       string       `json:"secret"`
	TriggerCount int32        `json:"trigger_count"`
	State        WebhookState `json:"state"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
