package agent

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const defaultMaxSteps = 50

var (
	ErrMaxSteps        = msg.ErrMaxSteps
	ErrAborted         = msg.ErrAborted
	ErrToolNotFound    = msg.ErrToolNotFound
	ErrEmptyContext    = msg.ErrEmptyContext
	ErrInvalidContinue = msg.ErrInvalidContinue
)

// DefaultConfig returns conservative defaults for a new agent run.
func DefaultConfig() Config {
	return Config{
		MaxSteps:      defaultMaxSteps,
		ToolExecution: ToolExecutionParallel,
		SteeringMode:  QueueAll,
		FollowUpMode:  QueueAll,
	}
}

// NewUserMessage builds a text user message with the current timestamp.
func NewUserMessage(text string) UserMessage {
	return UserMessage{
		Parts:     []ContentPart{TextPart{Text: text}},
		Timestamp: time.Now().UTC(),
	}
}

// NewUserMessageWithParts builds a multimodal user message.
func NewUserMessageWithParts(parts ...ContentPart) UserMessage {
	return UserMessage{
		Parts:     parts,
		Timestamp: time.Now().UTC(),
	}
}
