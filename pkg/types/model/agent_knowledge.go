package model

import "time"

// AgentKnowledge represents one knowledge-base markdown document for UI and transport.
type AgentKnowledge struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
