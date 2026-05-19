package model

import "time"

type EventOutbox struct {
	ID        int64     `json:"id"`
	EventID   string    `json:"event_id"`
	Payload   JSON      `json:"payload"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"created_at"`
}
