package coding_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolMetadata(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ws := coding.Workspace{Root: root}
	tools := []interface {
		Name() string
		Description() string
		Parameters() map[string]any
	}{
		coding.RunTerminalTool{Workspace: ws},
		coding.ReadFileTool{Workspace: ws},
		coding.WriteFileTool{Workspace: ws},
		coding.WebSearchTool{},
		coding.RunCodeTool{Workspace: ws},
	}

	tests := []struct {
		name string
	}{
		{name: "each tool exposes non-empty name"},
		{name: "each tool exposes description"},
		{name: "each tool exposes object parameters schema"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, tool := range tools {
				require.NotEmpty(t, tool.Name())
				require.NotEmpty(t, tool.Description())
				params := tool.Parameters()
				require.Equal(t, "object", params["type"])
				require.NotEmpty(t, params["properties"])
			}
		})
	}
}

func TestWorkspaceTruncateOutputDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{name: "default limit truncates large output", input: string(make([]byte, 9000))},
		{name: "default limit keeps small output", input: "ok"},
		{name: "default limit adds marker when truncated", input: string(make([]byte, 9000))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ws := coding.Workspace{}
			got := ws.TruncateOutput(tt.input)
			if len(tt.input) > 8192 {
				assert.Contains(t, got, "truncated")
				return
			}
			assert.Equal(t, tt.input, got)
		})
	}
}
