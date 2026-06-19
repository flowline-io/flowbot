package chatagent

import (
	"github.com/bytedance/sonic"
)

// Stream event type constants for Chat Agent SSE clients.
const (
	EventTypeDelta           = "delta"
	EventTypeTool            = "tool"
	EventTypeUsage           = "usage"
	EventTypeConfirm         = "confirm"
	EventTypeConfirmResolved = "confirm_resolved"
	EventTypeModeChange      = "mode_change"
	EventTypeCanceled        = "canceled"
	EventTypeDone            = "done"
	EventTypeError           = "error"
)

// StreamEvent is one SSE payload emitted to Chat Agent HTTP clients.
type StreamEvent struct {
	Type string `json:"type"`

	// delta / done
	Text string `json:"text,omitempty"`

	// tool
	Name     string `json:"name,omitempty"`
	Subagent string `json:"subagent,omitempty"`
	Status   string `json:"status,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`

	// usage
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	ContextPercent   float64 `json:"context_percent,omitempty"`
	ContextWindow    int     `json:"context_window,omitempty"`

	// confirm / confirm_resolved
	ID               string `json:"id,omitempty"`
	Tool             string `json:"tool,omitempty"`
	Summary          string `json:"summary,omitempty"`
	Permission       string `json:"permission,omitempty"`
	Pattern          string `json:"pattern,omitempty"`
	SuggestedPattern string `json:"suggested_pattern,omitempty"`
	SuggestAlways    bool   `json:"suggest_always,omitempty"`
	Approved         bool   `json:"approved,omitempty"`
	Reason           string `json:"reason,omitempty"`
	Mode             string `json:"mode,omitempty"`
	Message          string `json:"message,omitempty"`
}

// EventPublisher delivers stream events to an active HTTP SSE connection.
type EventPublisher interface {
	Publish(event StreamEvent) error
}

// MarshalStreamEvent serializes one SSE data frame body.
func MarshalStreamEvent(event StreamEvent) (string, error) {
	data, err := sonic.MarshalString(event)
	if err != nil {
		return "", err
	}
	return data, nil
}

// FormatSSEData returns a complete SSE data line payload for writing to the stream.
func FormatSSEData(event StreamEvent) (string, error) {
	body, err := MarshalStreamEvent(event)
	if err != nil {
		return "", err
	}
	return "data: " + body + "\n\n", nil
}
