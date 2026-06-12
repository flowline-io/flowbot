package chatagent

import "errors"

// API error sentinels for Chat Agent HTTP handlers.
var (
	// ErrConfirmNotFound means no pending confirmation exists for the session.
	ErrConfirmNotFound = errors.New("confirm not found")
	// ErrConfirmResolved means the confirmation was already resolved.
	ErrConfirmResolved = errors.New("confirm already resolved")
	// ErrChatAgentDisabled means chat agent is not configured.
	ErrChatAgentDisabled = errors.New("chat agent disabled")
	// ErrRunInFlight means the session already has an active SSE run.
	ErrRunInFlight = errors.New("run already in progress")
)
