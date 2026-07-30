package loop

import "github.com/flowline-io/flowbot/pkg/agent/msg"

const defaultMaxSteps = 50

var (
	// ErrMaxSteps is returned when the loop exceeds msg.Config.MaxSteps.
	ErrMaxSteps = msg.ErrMaxSteps
	// ErrAborted is returned when the loop is cancelled.
	ErrAborted = msg.ErrAborted
	// ErrToolNotFound is returned when a tool call names an unknown tool.
	ErrToolNotFound = msg.ErrToolNotFound
	// ErrEmptyContext is returned when Continue is called with no messages.
	ErrEmptyContext = msg.ErrEmptyContext
	// ErrInvalidContinue is returned when Continue cannot resume from the last message.
	ErrInvalidContinue = msg.ErrInvalidContinue
)

// DefaultConfig returns conservative defaults for a new agent run.
func DefaultConfig() msg.Config {
	return msg.Config{
		MaxSteps:      defaultMaxSteps,
		ToolExecution: msg.ToolExecutionParallel,
		SteeringMode:  msg.QueueAll,
		FollowUpMode:  msg.QueueAll,
	}
}

// NewUserMessage builds a text user message with the current timestamp.
func NewUserMessage(text string) msg.UserMessage {
	return msg.NewUserMessage(text)
}

// NewUserMessageWithParts builds a multimodal user message.
func NewUserMessageWithParts(parts ...msg.ContentPart) msg.UserMessage {
	return msg.NewUserMessageWithParts(parts...)
}
