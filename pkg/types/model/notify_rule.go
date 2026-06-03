package model

import "time"

// NotifyRule represents a notification routing rule for UI display and editing.
type NotifyRule struct {
	ID             int64     `json:"id"`
	RuleID         string    `json:"rule_id"`
	Name           string    `json:"name"`
	Action         string    `json:"action"`
	EventPattern   string    `json:"event_pattern"`
	ChannelPattern string    `json:"channel_pattern"`
	Condition      string    `json:"condition"`
	Priority       int       `json:"priority"`
	ParamsJSON     string    `json:"params_json"` // JSON string for form display
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
