package mcp

import (
	"errors"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ToolError is the machine-readable MCP tool failure stored at structuredContent.error.
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

// ToToolError maps err onto the MCP tool error object.
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

// Object returns the JSON object for structuredContent.error.
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
}

func codeRules() []codeRule {
	return []codeRule{
		{types.ErrInvalidArgument, "invalid_argument", false},
		{types.ErrUnauthorized, "unauthenticated", false},
		{types.ErrForbidden, "permission_denied", false},
		{types.ErrNotFound, "not_found", false},
		{types.ErrAlreadyExists, "already_exists", false},
		{types.ErrConflict, "conflict", true},
		{types.ErrRateLimited, "rate_limited", true},
		{types.ErrUnavailable, "unavailable", true},
		{types.ErrTimeout, "timeout", true},
		{types.ErrNotImplemented, "not_implemented", false},
		{types.ErrProvider, "provider_error", false},
		{types.ErrInvokeRun, "run_failed", false},
		{types.ErrInternal, "internal", false},
	}
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
	switch code {
	case "invalid_argument":
		if field != "" {
			return "Fix field " + field + "."
		}
		return "Fix the arguments."
	case "unauthenticated":
		return "Token missing or expired."
	case "permission_denied":
		return "Token lacks the required scope."
	case "not_found":
		return "Call the matching list or get tool, then retry."
	case "conflict":
		return "Read the current value, then write again."
	case "rate_limited":
		return "Wait retry_after_ms, then retry the same call."
	case "unavailable", "timeout":
		return "Retry with backoff."
	case "not_implemented":
		return "This operation is not available in the current configuration."
	default:
		return ""
	}
}
