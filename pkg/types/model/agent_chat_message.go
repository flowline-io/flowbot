package model

import "time"

// AgentChatMessage is one user or assistant turn for chat UI display.
type AgentChatMessage struct {
	Role               string    `json:"role"`
	Kind               string    `json:"kind"`
	Text               string    `json:"text"`
	HTML               string    `json:"html"`
	ToolName           string    `json:"tool_name"`
	ToolStatus         string    `json:"tool_status"`
	ToolStdout         string    `json:"tool_stdout"`
	ToolStderr         string    `json:"tool_stderr"`
	DurationMs         int64     `json:"duration_ms,omitempty"`
	TurnDurationMs     int64     `json:"turn_duration_ms,omitempty"`
	ThinkingDurationMs int64     `json:"thinking_duration_ms,omitempty"`
	RunDurationMs      int64     `json:"run_duration_ms,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}
