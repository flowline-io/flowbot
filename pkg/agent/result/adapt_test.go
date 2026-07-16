package result_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

func TestFileResultOrSessionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    result.Result[string, result.FileError]
		message  string
		wantVal  string
		wantErr  bool
		wantCode string
	}{
		{
			name:    "ok result returns value",
			input:   result.Ok[string, result.FileError]("payload"),
			message: "read file",
			wantVal: "payload",
		},
		{
			name: "not_found maps to not_found session error",
			input: result.Err[string, result.FileError](
				result.NewFileError("not_found", "missing", nil),
			),
			message:  "load session",
			wantErr:  true,
			wantCode: "not_found",
		},
		{
			name: "other file error maps to storage session error",
			input: result.Err[string, result.FileError](
				result.NewFileError("permission_denied", "denied", errors.New("acl")),
			),
			message:  "write session",
			wantErr:  true,
			wantCode: "storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, err := result.FileResultOrSessionError(tt.input, tt.message)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, result.CodeOf(err))
				assert.Contains(t, err.Error(), tt.message)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestToHarnessError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		subsystem string
		message   string
		cause     error
		wantCode  string
	}{
		{
			name:      "wraps cause as harness error",
			subsystem: "compaction",
			message:   "failed to compact",
			cause:     errors.New("llm timeout"),
			wantCode:  "compaction",
		},
		{
			name:      "nil cause still builds harness error",
			subsystem: "session",
			message:   "missing branch",
			cause:     nil,
			wantCode:  "session",
		},
		{
			name:      "tool subsystem code",
			subsystem: "tool",
			message:   "permission denied",
			cause:     errors.New("denied"),
			wantCode:  "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := result.ToHarnessError(tt.subsystem, tt.message, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, result.CodeOf(err))
			assert.Contains(t, err.Error(), tt.message)
			if tt.cause != nil {
				assert.ErrorIs(t, err, tt.cause)
			}
		})
	}
}

func TestResultErrorValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		build    func() result.Result[int, result.ParseError]
		wantCode string
	}{
		{
			name: "error value from failed result",
			build: func() result.Result[int, result.ParseError] {
				return result.Err[int, result.ParseError](result.NewParseError("json", "bad", nil))
			},
			wantCode: "json",
		},
		{
			name: "error value from ok result is zero",
			build: func() result.Result[int, result.ParseError] {
				return result.Ok[int, result.ParseError](1)
			},
			wantCode: "",
		},
		{
			name: "error value from zero result",
			build: func() result.Result[int, result.ParseError] {
				return result.Result[int, result.ParseError]{}
			},
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tt.build()
			assert.Equal(t, tt.wantCode, r.ErrorValue().Code())
		})
	}
}
