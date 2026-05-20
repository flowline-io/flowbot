package model

import "time"

type DataEvent struct {
	ID             int64     `json:"id"`
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Source         string    `json:"source"`
	Capability     string    `json:"capability"`
	Operation      string    `json:"operation"`
	Backend        string    `json:"backend"`
	App            string    `json:"app"`
	EntityID       string    `json:"entity_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	UID            string    `json:"uid"`
	Topic          string    `json:"topic"`
	Data           JSON      `json:"data"`
	CreatedAt      time.Time `json:"created_at"`
}
