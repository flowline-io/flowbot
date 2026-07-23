package model

import "time"

// AgentMemoryFact is one keyed fact in a memory scope for UI and transport.
type AgentMemoryFact struct {
	ID        int64     `json:"id"`
	Scope     string    `json:"scope"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
