package idempotency

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

const retryInProgress = time.Second

// ErrorFor maps a non-run outcome onto the agent-facing domain error.
func ErrorFor(outcome Outcome) error {
	switch outcome {
	case OutcomeInProgress:
		return &types.Error{
			Kind:       types.ErrUnavailable,
			Message:    "idempotency key is already in progress",
			RetryAfter: retryInProgress,
		}
	case OutcomeUnknown:
		return types.Errorf(types.ErrIdempotencyUnknown, "idempotency key outcome is unknown")
	case OutcomeMismatch:
		return &types.Error{
			Kind:    types.ErrInvalidArgument,
			Message: "idempotency key was already used with different arguments",
			Field:   "idempotency_key",
		}
	default:
		return nil
	}
}
