package result

import (
	"errors"
	"fmt"
)

// CodedError is implemented by typed agent errors with stable code strings.
type CodedError interface {
	error
	Code() string
}

// FileError describes expected filesystem failures.
type FileError struct {
	code    string
	Message string
	Cause   error
}

func (e FileError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("file %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("file %s: %s", e.code, e.Message)
}

func (e FileError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e FileError) Code() string { return e.code }

// ExecutionError describes expected shell/process failures.
type ExecutionError struct {
	code    string
	Message string
	Cause   error
}

func (e ExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("execution %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("execution %s: %s", e.code, e.Message)
}

func (e ExecutionError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e ExecutionError) Code() string { return e.code }

// CompactionError describes expected context compaction failures.
type CompactionError struct {
	code    string
	Message string
	Cause   error
}

func (e CompactionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("compaction %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("compaction %s: %s", e.code, e.Message)
}

func (e CompactionError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e CompactionError) Code() string { return e.code }

// BranchSummaryError describes expected branch summarization failures.
type BranchSummaryError struct {
	code    string
	Message string
	Cause   error
}

func (e BranchSummaryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("branch_summary %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("branch_summary %s: %s", e.code, e.Message)
}

func (e BranchSummaryError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e BranchSummaryError) Code() string { return e.code }

// SessionError describes expected session persistence failures.
type SessionError struct {
	code    string
	Message string
	Cause   error
}

func (e SessionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("session %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("session %s: %s", e.code, e.Message)
}

func (e SessionError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e SessionError) Code() string { return e.code }

// ParseError describes JSONL or message payload parse failures.
type ParseError struct {
	code    string
	Message string
	Cause   error
}

func (e ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("parse %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("parse %s: %s", e.code, e.Message)
}

func (e ParseError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e ParseError) Code() string { return e.code }

// OverflowError indicates the model context window was exceeded.
type OverflowError struct {
	Message string
	Cause   error
}

func (e OverflowError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("context overflow: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("context overflow: %s", e.Message)
}

func (e OverflowError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (OverflowError) Code() string { return "overflow" }

// HarnessError normalizes subsystem failures at the harness public API boundary.
type HarnessError struct {
	code    string
	Message string
	Cause   error
}

func (e HarnessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("harness %s: %s: %v", e.code, e.Message, e.Cause)
	}
	return fmt.Sprintf("harness %s: %s", e.code, e.Message)
}

func (e HarnessError) Unwrap() error { return e.Cause }

// Code returns the stable error code.
func (e HarnessError) Code() string { return e.code }

// NewFileError builds a FileError with the given code and message.
func NewFileError(code, message string, cause error) FileError {
	return FileError{code: code, Message: message, Cause: cause}
}

// NewExecutionError builds an ExecutionError with the given code and message.
func NewExecutionError(code, message string, cause error) ExecutionError {
	return ExecutionError{code: code, Message: message, Cause: cause}
}

// NewCompactionError builds a CompactionError with the given code and message.
func NewCompactionError(code, message string, cause error) CompactionError {
	return CompactionError{code: code, Message: message, Cause: cause}
}

// NewBranchSummaryError builds a BranchSummaryError with the given code and message.
func NewBranchSummaryError(code, message string, cause error) BranchSummaryError {
	return BranchSummaryError{code: code, Message: message, Cause: cause}
}

// NewSessionError builds a SessionError with the given code and message.
func NewSessionError(code, message string, cause error) SessionError {
	return SessionError{code: code, Message: message, Cause: cause}
}

// NewParseError builds a ParseError with the given code and message.
func NewParseError(code, message string, cause error) ParseError {
	return ParseError{code: code, Message: message, Cause: cause}
}

// NewHarnessError builds a HarnessError with the given code and message.
func NewHarnessError(code, message string, cause error) HarnessError {
	return HarnessError{code: code, Message: message, Cause: cause}
}

// NewOverflowError builds an OverflowError wrapping an underlying cause.
func NewOverflowError(message string, cause error) OverflowError {
	return OverflowError{Message: message, Cause: cause}
}

// CodeOf returns the stable code for a typed agent error, or empty string.
func CodeOf(err error) string {
	var coded CodedError
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return ""
}

// IsCode reports whether err is a CodedError with the given code.
func IsCode(err error, code string) bool {
	return CodeOf(err) == code
}
