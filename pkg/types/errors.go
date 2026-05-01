package types

import (
	"errors"
	"fmt"
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

func (e *Error) Error() string {
	if e == nil {
		return "error"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "error"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	return errors.Is(e.Kind, target)
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
