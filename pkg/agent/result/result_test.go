package result_test

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResultOkErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		build     func() result.Result[string, result.FileError]
		wantOk    bool
		wantValue string
		wantCode  string
	}{
		{
			name:      "ok returns value",
			build:     func() result.Result[string, result.FileError] { return result.Ok[string, result.FileError]("hello") },
			wantOk:    true,
			wantValue: "hello",
		},
		{
			name: "err returns typed failure",
			build: func() result.Result[string, result.FileError] {
				return result.Err[string, result.FileError](result.NewFileError("not_found", "missing", nil))
			},
			wantOk:   false,
			wantCode: "not_found",
		},
		{
			name: "zero value is not ok",
			build: func() result.Result[string, result.FileError] {
				return result.Result[string, result.FileError]{}
			},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tt.build()
			assert.Equal(t, tt.wantOk, r.IsOk())
			if tt.wantOk {
				assert.Equal(t, tt.wantValue, r.Value())
				return
			}
			_, errVal, ok := r.ValueOrZero()
			assert.False(t, ok)
			assert.Equal(t, tt.wantCode, errVal.Code())
		})
	}
}

func TestGetOrError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    result.Result[int, result.CompactionError]
		wantVal  int
		wantErr  bool
		wantCode string
	}{
		{
			name:    "success value",
			input:   result.Ok[int, result.CompactionError](42),
			wantVal: 42,
		},
		{
			name: "failure error",
			input: result.Err[int, result.CompactionError](
				result.NewCompactionError("summarization_failed", "llm failed", errors.New("upstream")),
			),
			wantErr:  true,
			wantCode: "summarization_failed",
		},
		{
			name:     "aborted compaction",
			input:    result.Err[int, result.CompactionError](result.NewCompactionError("aborted", "cancelled", nil)),
			wantErr:  true,
			wantCode: "aborted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, err := result.GetOrError(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, result.CodeOf(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestTypedErrorUnwrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{
			name:     "file error code",
			err:      result.NewFileError("permission_denied", "denied", nil),
			wantCode: "permission_denied",
		},
		{
			name:     "execution error code",
			err:      result.NewExecutionError("timeout", "timed out", nil),
			wantCode: "timeout",
		},
		{
			name:     "harness error code",
			err:      result.NewHarnessError("compaction", "failed", nil),
			wantCode: "compaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantCode, result.CodeOf(tt.err))
			assert.Error(t, tt.err)
		})
	}
}
