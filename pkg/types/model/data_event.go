package model

import "time"

// DataEvent is a durable business event row for UI tables.
type DataEvent struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	Source     string         `json:"source"`
	Capability string         `json:"capability,omitempty"`
	EntityID   string         `json:"entity_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	Data       map[string]any `json:"data,omitempty"`
}

// NotificationRecord is a notification delivery history row for UI and gateway use.
type NotificationRecord struct {
	ID              int64          `json:"id"`
	UID             string         `json:"uid,omitempty"`
	Channel         string         `json:"channel"`
	RuleID          string         `json:"rule_id,omitempty"`
	TemplateID      string         `json:"template_id,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Status          string         `json:"status"`
	ErrorMsg        string         `json:"error_msg,omitempty"`
	PayloadSnapshot map[string]any `json:"payload_snapshot,omitempty"`
	CorrelationID   string         `json:"correlation_id,omitempty"`
	EscalateAt      *time.Time     `json:"escalate_at,omitempty"`
	ReadAt          *time.Time     `json:"read_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}
