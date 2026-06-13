package model

import "time"

// AgentSkill represents an agent skill definition for UI display and transport.
type AgentSkill struct {
	Flag                   string    `json:"flag"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	Content                string    `json:"content"`
	BaseDir                string    `json:"base_dir"`
	Source                 string    `json:"source"`
	Enabled                bool      `json:"enabled"`
	DisableModelInvocation bool      `json:"disable_model_invocation"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
