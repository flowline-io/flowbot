package msg

import "errors"

var (
	ErrMaxSteps        = errors.New("agent: max steps exceeded")
	ErrAborted         = errors.New("agent: aborted")
	ErrToolNotFound    = errors.New("agent: tool not found")
	ErrEmptyContext    = errors.New("agent: empty context")
	ErrInvalidContinue = errors.New("agent: cannot continue from assistant message")
)
