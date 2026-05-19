package model

import (
	"time"
)

// RateLimit mapped from table <rate_limits>
type RateLimit struct {
	ID         int64         `json:"id"`
	FlowID     *int64        `json:"flow_id"`
	NodeID     string        `json:"node_id"`
	LimitType  RateLimitType `json:"limit_type"`
	LimitValue int           `json:"limit_value"`
	WindowSize int           `json:"window_size"`
	WindowUnit string        `json:"window_unit"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}
