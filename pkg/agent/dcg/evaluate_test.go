package dcg_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateToolCall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		checker dcg.Checker
		want    dcg.ToolVerdict
	}{
		{
			name: "unguarded tool skips",
			tool: "read_file",
			args: map[string]any{"path": "a.go"},
			want: dcg.ToolVerdict{Skip: true},
		},
		{
			name:    "terminal allowed",
			tool:    "run_terminal",
			args:    map[string]any{"command": "ls"},
			checker: dcg.AllowAllChecker{},
			want:    dcg.ToolVerdict{Command: "ls"},
		},
		{
			name:    "terminal blocked",
			tool:    "run_terminal",
			args:    map[string]any{"command": "rm -rf /"},
			checker: dcg.DenyChecker{Reason: "nope"},
			want:    dcg.ToolVerdict{Block: true, Reason: "nope", Command: "rm -rf /"},
		},
		{
			name: "missing command blocks",
			tool: "run_terminal",
			args: map[string]any{},
			want: dcg.ToolVerdict{Block: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := dcg.EvaluateToolCall(context.Background(), tt.tool, tt.args, tt.checker)
			require.NoError(t, err)
			assert.Equal(t, tt.want.Skip, got.Skip)
			assert.Equal(t, tt.want.Block, got.Block)
			if tt.want.Reason != "" {
				assert.Equal(t, tt.want.Reason, got.Reason)
			}
			if tt.want.Block && tt.want.Reason == "" {
				assert.NotEmpty(t, got.Reason)
			}
			if tt.want.Command != "" {
				assert.Equal(t, tt.want.Command, got.Command)
			}
		})
	}
}
