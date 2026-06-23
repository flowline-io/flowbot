package model

import "time"

// AgentSubagent represents a chat assistant subagent definition for UI display and transport.
type AgentSubagent struct {
	Flag         string    `json:"flag"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Tools        []string  `json:"tools"`
	Skills       []string  `json:"skills"`
	Model        string    `json:"model"`
	Source       string    `json:"source"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentSubagentSkillOption is one selectable skill in the subagent form.
type AgentSubagentSkillOption struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AgentSubagentFormParams carries data for rendering the subagent create/edit form.
type AgentSubagentFormParams struct {
	Item            AgentSubagent              `json:"item"`
	IsNew           bool                       `json:"is_new"`
	Errors          map[string]string          `json:"errors,omitempty"`
	AvailableTools  []string                   `json:"available_tools"`
	AvailableSkills []AgentSubagentSkillOption `json:"available_skills"`
}

// AgentSubagentTask represents one delegated subagent task for UI display and transport.
type AgentSubagentTask struct {
	ID           int64      `json:"id"`
	SessionID    string     `json:"session_id"`
	SubagentName string     `json:"subagent_name"`
	Description  string     `json:"description"`
	Prompt       string     `json:"prompt"`
	Status       string     `json:"status"`
	Result       string     `json:"result"`
	ErrorText    string     `json:"error_text"`
	Depth        int        `json:"depth"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
