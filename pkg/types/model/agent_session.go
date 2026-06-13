package model

import "time"

// AgentSession represents a chat agent session for UI display and transport.
type AgentSession struct {
	Flag      string    `json:"flag"`
	UID       string    `json:"uid"`
	LeafID    string    `json:"leaf_id"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
