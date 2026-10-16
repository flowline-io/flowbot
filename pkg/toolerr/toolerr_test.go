package toolerr

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestToToolErrorSentinels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		code      string
		retryable bool
		hint      string
		message   string
	}{
		{
			name:    "invalid argument",
			err:     types.Errorf(types.ErrInvalidArgument, "name is required"),
			code:    "invalid_argument",
			hint:    "Fix the arguments.",
			message: "name is required",
		},
		{
			name:    "invalid argument with field",
			err:     &types.Error{Kind: types.ErrInvalidArgument, Message: "name is required", Field: "name"},
			code:    "invalid_argument",
			hint:    "Fix field name.",
			message: "name is required",
		},
		{
			name:    "invalid argument keeps cause",
			err:     types.WrapError(types.ErrInvalidArgument, "invalid workflow YAML", errors.New("bad indent")),
			code:    "invalid_argument",
			hint:    "Fix the arguments.",
			message: "invalid workflow YAML: bad indent",
		},
		{
			name:    "unauthorized",
			err:     types.ErrUnauthorized,
			code:    "unauthenticated",
			hint:    "Token missing or expired.",
			message: "unauthorized",
		},
		{
			name:    "forbidden",
			err:     types.Errorf(types.ErrForbidden, "missing scope"),
			code:    "permission_denied",
			hint:    "Token lacks the required scope.",
			message: "missing scope",
		},
		{
			name:    "not found",
			err:     types.ErrNotFound,
			code:    "not_found",
			hint:    "Call the matching list or get tool, then retry.",
			message: "not found",
		},
		{
			name:    "already exists",
			err:     types.ErrAlreadyExists,
			code:    "already_exists",
			message: "already exists",
		},
		{
			name:      "conflict",
			err:       types.Errorf(types.ErrConflict, "version mismatch"),
			code:      "conflict",
			retryable: true,
			hint:      "Read the current value, then write again.",
			message:   "version mismatch",
		},
		{
			name:      "rate limited",
			err:       &types.Error{Kind: types.ErrRateLimited, Message: "miniflux API rate limited", RetryAfter: 5 * time.Second},
			code:      "rate_limited",
			retryable: true,
			hint:      "Wait retry_after_ms, then retry the same call.",
			message:   "miniflux API rate limited",
		},
		{
			name:      "unavailable",
			err:       types.Errorf(types.ErrUnavailable, "pipeline service not ready"),
			code:      "unavailable",
			retryable: true,
			hint:      "Retry with backoff.",
			message:   "pipeline service not ready",
		},
		{
			name:      "timeout",
			err:       types.ErrTimeout,
			code:      "timeout",
			retryable: true,
			hint:      "Retry with backoff.",
			message:   "timeout",
		},
		{
			name:    "not implemented",
			err:     types.Errorf(types.ErrNotImplemented, "runtime is noop"),
			code:    "not_implemented",
			hint:    "This operation is not available in the current configuration.",
			message: "runtime is noop",
		},
		{
			name:      "provider retryable",
			err:       &types.Error{Kind: types.ErrProvider, Message: "upstream call failed", Retryable: true, Cause: errors.New("dsn=secret")},
			code:      "provider_error",
			retryable: true,
			message:   "upstream call failed",
		},
		{
			name:    "provider not retryable hides cause",
			err:     types.WrapError(types.ErrProvider, "upstream call failed", errors.New("dsn=secret")),
			code:    "provider_error",
			message: "upstream call failed",
		},
		{
			name:    "internal",
			err:     types.ErrInternal,
			code:    "internal",
			message: "internal error",
		},
		{
			name:    "internal with message stays opaque",
			err:     &types.Error{Kind: types.ErrInternal, Message: "postgres: password denied"},
			code:    "internal",
			message: "internal error",
		},
		{
			name:    "invoke run",
			err:     types.Errorf(types.ErrInvokeRun, "function exited 1"),
			code:    "run_failed",
			message: "function exited 1",
		},
		{
			name:    "plain error stays opaque",
			err:     errors.New("postgres: password denied"),
			code:    "internal",
			message: "internal error",
		},
		{
			name:    "nil",
			code:    "internal",
			message: "internal error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToToolError(tt.err)
			assert.Equal(t, tt.code, got.Code)
			assert.Equal(t, tt.retryable, got.Retryable)
			assert.Equal(t, tt.hint, got.Hint)
			assert.Equal(t, tt.message, got.Message)
			obj := got.Object()
			assert.Equal(t, tt.code, obj["code"])
			assert.Contains(t, obj, "hint")
			assert.NotContains(t, obj, "request_id")
			if tt.err != nil {
				assert.NotContains(t, fmt.Sprint(obj["message"]), "postgres")
				assert.NotContains(t, fmt.Sprint(obj["message"]), "dsn=secret")
			}
		})
	}
}

func TestToToolErrorOptionalKeys(t *testing.T) {
	t.Parallel()
	got := ToToolError(&types.Error{
		Kind:       types.ErrRateLimited,
		Message:    "slow down",
		Field:      "limit",
		RetryAfter: 5 * time.Second,
	})
	obj := got.Object()
	assert.Equal(t, int64(5000), obj["retry_after_ms"])
	assert.Equal(t, "limit", obj["field"])
	assert.NotContains(t, obj, "job_id")
	assert.NotContains(t, obj, "confirmation_required")

	short := ToToolError(&types.Error{Kind: types.ErrRateLimited, Message: "slow down", RetryAfter: time.Microsecond})
	assert.NotContains(t, short.Object(), "retry_after_ms")

	set := ToToolError(&types.Error{
		Kind:         types.ErrConflict,
		Message:      "stale",
		JobID:        "job-1",
		Confirmation: map[string]any{"token": "abc"},
	})
	setObj := set.Object()
	assert.Equal(t, "job-1", setObj["job_id"])
	assert.Equal(t, map[string]any{"token": "abc"}, setObj["confirmation_required"])
}
