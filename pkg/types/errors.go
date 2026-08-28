package types

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrConflict        = errors.New("conflict")
	ErrRateLimited     = errors.New("rate limited")
	ErrUnavailable     = errors.New("unavailable")
	ErrTimeout         = errors.New("timeout")
	ErrNotImplemented  = errors.New("not implemented")
	ErrProvider        = errors.New("provider error")
	ErrInternal        = errors.New("internal error")
)

// Error carries machine-readable domain error metadata across ability, hub,
// pipeline, workflow, and HTTP boundaries.
type Error struct {
	Kind       error
	Code       string
	Message    string
	Capability string
	Operation  string
	Provider   string
	Retryable  bool
	Cause      error
}

// Error returns the operator-facing text: Message joined with Cause when both are set.
// HTTP clients use Message via protocol.NewFailedResponse, not this string.
func (e *Error) Error() string {
	if e == nil {
		return "error"
	}
	msg := e.Message
	if msg == "" && e.Kind != nil {
		msg = e.Kind.Error()
	}
	cause := ""
	if e.Cause != nil {
		cause = e.Cause.Error()
	}
	if text := joinErrorText(msg, cause); text != "" {
		return text
	}
	return "error"
}

func joinErrorText(message, cause string) string {
	switch {
	case message == "":
		return cause
	case cause == "" || cause == message:
		return message
	case strings.HasSuffix(message, cause):
		return message
	default:
		return message + ": " + cause
	}
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// ClientMessage returns the domain Message without Cause, for HTTP JSON bodies.
// Non-domain errors fall back to err.Error().
func ClientMessage(err error) string {
	if err == nil {
		return ""
	}
	var de *Error
	if !errors.As(err, &de) {
		return err.Error()
	}
	if de == nil {
		return err.Error()
	}
	if de.Message != "" {
		return de.Message
	}
	if de.Kind != nil {
		return de.Kind.Error()
	}
	return err.Error()
}

func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	return errors.Is(e.Kind, target)
}

// RetryableCode returns the error code for retry filtering.
func (e *Error) RetryableCode() string {
	if e == nil {
		return ""
	}
	return e.Code
}

// IsRetryableError returns true if the error is marked as retryable.
func (e *Error) IsRetryableError() bool {
	if e == nil {
		return false
	}
	return e.Retryable
}

// WrapError wraps a lower-level cause with a standard Flowbot error kind.
func WrapError(kind error, message string, cause error) error {
	return &Error{
		Kind:    kind,
		Message: message,
		Cause:   cause,
	}
}

// Errorf creates a standard Flowbot error with a formatted message.
func Errorf(kind error, format string, args ...any) error {
	return &Error{
		Kind:    kind,
		Message: fmt.Sprintf(format, args...),
	}
}
