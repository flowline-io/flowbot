package model

import "time"

// AgentSubagent represents a chat assistant subagent definition for UI display and transport.
type AgentSubagent struct {
	Flag         string    `json:"flag"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Tools        []string  `json:"tools"`
	Model        string    `json:"model"`
	Source       string    `json:"source"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
