// Package toolerr maps domain errors onto the agent-facing error object
// shared by MCP tools and the flowbot CLI.
package toolerr

import (
	"errors"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ToolError is the machine-readable failure stored at error in MCP structuredContent
// and on the CLI stderr envelope.
type ToolError struct {
	Code         string
	Message      string
	Retryable    bool
	Hint         string
	RetryAfterMS int64
	Field        string
	JobID        string
	Confirmation map[string]any
}

// ToToolError maps err onto the shared tool error object.
// Plain errors and ErrInternal become code internal with a fixed message.
// ErrInvalidArgument keeps the cause text from types.ClientSafeDomainMessage.
func ToToolError(err error) ToolError {
	code, retryable := toolCode(err)
	var te *types.Error
	if errors.As(err, &te) && te != nil {
		retryable = toolRetryable(code, te, retryable)
	}
	out := ToolError{
		Code:      code,
		Message:   toolMessage(err, te, code),
		Retryable: retryable,
		Hint:      toolHint(code, fieldOf(te)),
	}
	if te != nil {
		if te.Field != "" {
			out.Field = te.Field
		}
		if ms := te.RetryAfter.Milliseconds(); ms > 0 {
			out.RetryAfterMS = ms
		}
		if te.JobID != "" {
			out.JobID = te.JobID
		}
		if len(te.Confirmation) > 0 {
			out.Confirmation = te.Confirmation
		}
	}
	return out
}

// Object returns the JSON object for an error envelope.
// Optional keys are omitted when unset. hint is always present.
func (e ToolError) Object() map[string]any {
	obj := map[string]any{
		"code":      e.Code,
		"message":   e.Message,
		"retryable": e.Retryable,
		"hint":      e.Hint,
	}
	if e.RetryAfterMS > 0 {
		obj["retry_after_ms"] = e.RetryAfterMS
	}
	if e.Field != "" {
		obj["field"] = e.Field
	}
	if e.JobID != "" {
		obj["job_id"] = e.JobID
	}
	if len(e.Confirmation) > 0 {
		obj["confirmation_required"] = e.Confirmation
	}
	return obj
}

func fieldOf(te *types.Error) string {
	if te == nil {
		return ""
	}
	return te.Field
}

func toolRetryable(code string, te *types.Error, fallback bool) bool {
	if code == "provider_error" && te != nil {
		return te.Retryable
	}
	return fallback
}

type codeRule struct {
	err       error
	code      string
	retryable bool
	hint      string
	exit      int
}

func codeRules() []codeRule {
	return []codeRule{
		{types.ErrInvalidArgument, "invalid_argument", false, "Fix the arguments.", 2},
		{types.ErrUnauthorized, "unauthenticated", false, "Token missing or expired.", 3},
		{types.ErrForbidden, "permission_denied", false, "Token lacks the required scope.", 3},
		{types.ErrNotFound, "not_found", false, "Call the matching list or get tool, then retry.", 4},
		{types.ErrAlreadyExists, "already_exists", false, "", 5},
		{types.ErrConflict, "conflict", true, "Read the current value, then write again.", 5},
		{types.ErrRateLimited, "rate_limited", true, "Wait retry_after_ms, then retry the same call.", 6},
		{types.ErrUnavailable, "unavailable", true, "Retry with backoff.", 6},
		{types.ErrTimeout, "timeout", true, "Retry with backoff.", 6},
		{types.ErrNotImplemented, "not_implemented", false, "This operation is not available in the current configuration.", 8},
		{types.ErrProvider, "provider_error", false, "", 1},
		{types.ErrInvokeRun, "run_failed", false, "", 1},
		{types.ErrInternal, "internal", false, "", 1},
	}
}

// ProcessExit is the CLI process status for a tool error code.
// Retryable provider_error is 6. Every other unknown code is 1.
func ProcessExit(code string, retryable bool) int {
	if code == "provider_error" && retryable {
		return 6
	}
	for _, rule := range codeRules() {
		if rule.code == code {
			return rule.exit
		}
	}
	return 1
}

func toolCode(err error) (string, bool) {
	if err == nil {
		return "internal", false
	}
	for _, rule := range codeRules() {
		if errors.Is(err, rule.err) {
			return rule.code, rule.retryable
		}
	}
	return "internal", false
}

func toolMessage(err error, te *types.Error, code string) string {
	if code == "internal" || err == nil {
		return "internal error"
	}
	if te != nil {
		msg := types.ClientSafeDomainMessage(te)
		if msg != "" && msg != "Unknown Error" {
			return msg
		}
	}
	if bare := bareSentinel(err); bare != nil {
		return bare.Error()
	}
	return "internal error"
}

func bareSentinel(err error) error {
	if err == nil {
		return nil
	}
	for _, rule := range codeRules() {
		if errors.Is(err, rule.err) {
			return rule.err
		}
	}
	return nil
}

func toolHint(code, field string) string {
	if code == "invalid_argument" && field != "" {
		return "Fix field " + field + "."
	}
	for _, rule := range codeRules() {
		if rule.code == code {
			return rule.hint
		}
	}
	return ""
}
