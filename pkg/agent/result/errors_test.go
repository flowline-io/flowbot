package result_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

func TestTypedErrorConstructors(t *testing.T) {
	t.Parallel()

	cause := errors.New("root cause")

	tests := []struct {
		name     string
		err      error
		wantCode string
		wantMsg  string
	}{
		{
			name:     "file error with cause",
			err:      result.NewFileError("not_found", "missing file", cause),
			wantCode: "not_found",
			wantMsg:  "file not_found: missing file: root cause",
		},
		{
			name:     "execution error without cause",
			err:      result.NewExecutionError("timeout", "timed out", nil),
			wantCode: "timeout",
			wantMsg:  "execution timeout: timed out",
		},
		{
			name:     "compaction error with cause",
			err:      result.NewCompactionError("failed", "summarize failed", cause),
			wantCode: "failed",
			wantMsg:  "compaction failed: summarize failed: root cause",
		},
		{
			name:     "branch summary error",
			err:      result.NewBranchSummaryError("empty", "no messages", nil),
			wantCode: "empty",
			wantMsg:  "branch_summary empty: no messages",
		},
		{
			name:     "session error",
			err:      result.NewSessionError("storage", "write failed", cause),
			wantCode: "storage",
			wantMsg:  "session storage: write failed: root cause",
		},
		{
			name:     "parse error",
			err:      result.NewParseError("json", "invalid jsonl", nil),
			wantCode: "json",
			wantMsg:  "parse json: invalid jsonl",
		},
		{
			name:     "harness error",
			err:      result.NewHarnessError("llm", "model failed", cause),
			wantCode: "llm",
			wantMsg:  "harness llm: model failed: root cause",
		},
		{
			name:     "overflow error",
			err:      result.NewOverflowError("too large", cause),
			wantCode: "overflow",
			wantMsg:  "context overflow: too large: root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tt.err)
			assert.Equal(t, tt.wantCode, result.CodeOf(tt.err))
			assert.Equal(t, tt.wantMsg, tt.err.Error())
			assert.True(t, result.IsCode(tt.err, tt.wantCode))
			assert.False(t, result.IsCode(tt.err, "other"))
		})
	}
}

func TestTypedErrorCauseUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")

	tests := []struct {
		name string
		err  error
	}{
		{name: "file unwraps cause", err: result.NewFileError("io", "failed", cause)},
		{name: "execution unwraps cause", err: result.NewExecutionError("exit", "nonzero", cause)},
		{name: "session unwraps cause", err: result.NewSessionError("storage", "failed", cause)},
		{name: "parse unwraps cause", err: result.NewParseError("json", "bad", cause)},
		{name: "overflow unwraps cause", err: result.NewOverflowError("overflow", cause)},
		{name: "harness unwraps cause", err: result.NewHarnessError("tool", "failed", cause)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tt.err, cause)
		})
	}
}

func TestCodeOfAndIsCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode string
		wantIs   bool
	}{
		{
			name:     "coded error returns code",
			err:      result.NewFileError("permission_denied", "denied", nil),
			wantCode: "permission_denied",
			wantIs:   true,
		},
		{
			name:     "plain error returns empty code",
			err:      errors.New("plain"),
			wantCode: "",
			wantIs:   false,
		},
		{
			name:     "nil error returns empty code",
			err:      nil,
			wantCode: "",
			wantIs:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantCode, result.CodeOf(tt.err))
			assert.Equal(t, tt.wantIs, result.IsCode(tt.err, "permission_denied"))
		})
	}
}
