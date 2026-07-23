package model

import "time"

// AgentSessionSummary is a searchable summary derived from an archived chat session.
type AgentSessionSummary struct {
	ID          int64      `json:"id"`
	SessionFlag string     `json:"session_flag"`
	Scope       string     `json:"scope"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
}
