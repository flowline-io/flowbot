package model

import "time"

// AgentSession represents a chat agent session for UI display and transport.
type AgentSession struct {
	Flag   string `json:"flag"`
	Title  string `json:"title"`
	UID    string `json:"uid"`
	LeafID string `json:"leaf_id"`
	State  string `json:"state"`
	// Model holds the session-level chat model override (empty = global default).
	Model string `json:"model,omitempty"`
	// ThinkingLevel holds the session-level reasoning intensity (empty = default).
	ThinkingLevel   string            `json:"thinking_level,omitempty"`
	TotalDurationMs int64             `json:"total_duration_ms,omitempty"`
	TodoSummary     *AgentTodoSummary `json:"todo_summary,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AgentSessionEntry represents one append-only node in a chat session tree for UI display.
type AgentSessionEntry struct {
	Flag        string    `json:"flag"`
	SessionID   string    `json:"session_id"`
	ParentID    string    `json:"parent_id"`
	EntryType   string    `json:"entry_type"`
	PayloadJSON string    `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

// AgentPlan represents a persisted plan document for UI display.
type AgentPlan struct {
	PlanID    string    `json:"plan_id"`
	URI       string    `json:"uri"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
