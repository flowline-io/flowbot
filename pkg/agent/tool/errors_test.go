package tool_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatToolError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		code    string
		message string
		hint    string
		want    string
	}{
		{name: "with hint", code: "invalid_args", message: "missing path", hint: "provide path", want: "[invalid_args] missing path. Hint: provide path"},
		{name: "without hint", code: "io_error", message: "write failed", hint: "", want: "[io_error] write failed"},
		{name: "default code", code: "", message: "boom", hint: "", want: "[tool_error] boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tool.FormatToolError(tt.code, tt.message, tt.hint))
		})
	}
}

func TestValidateArgs(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"count":   map[string]any{"type": "number"},
			"enabled": map[string]any{"type": "boolean"},
		},
		"required": []any{"path"},
	}
	tests := []struct {
		name    string
		args    map[string]any
		wantErr string
	}{
		{name: "valid", args: map[string]any{"path": "a.go", "count": float64(1), "enabled": true}},
		{name: "missing required", args: map[string]any{}, wantErr: "missing required argument"},
		{name: "wrong type", args: map[string]any{"path": "a.go", "enabled": "yes"}, wantErr: "must be type boolean"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tool.ValidateArgs(schema, tt.args)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
