package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/memory"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func memoryToolResultText(result msg.ToolResultMessage) string {
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

func TestUpdateMemoryToolReadWriteList(t *testing.T) {
	LockAppConfigForTest(t)
	root := t.TempDir()
	memRoot := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{
		ChatModel: "gpt-test",
		Workspace: root,
	}

	store, err := memory.NewFileStore(memRoot, "MEMORIES.md", 4096)
	require.NoError(t, err)
	tool := UpdateMemoryTool{Store: store}

	tests := []struct {
		name     string
		scope    string
		args     map[string]any
		wantText string
		wantErr  bool
	}{
		{
			name: "write", scope: "pipe-a",
			args:     map[string]any{"operation": "write", "content": "Memory example"},
			wantText: "saved MEMORIES.md",
		},
		{
			name: "read", scope: "pipe-a",
			args:     map[string]any{"operation": "read"},
			wantText: "Memory example",
		},
		{
			name: "list", scope: "pipe-a",
			args:     map[string]any{"operation": "list"},
			wantText: "MEMORIES.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithMemoryScope(context.Background(), tt.scope)
			result, err := tool.Execute(ctx, "id-1", tt.args, nil)
			require.NoError(t, err)
			if tt.wantErr {
				assert.True(t, result.IsError)
				return
			}
			assert.False(t, result.IsError)
			assert.Contains(t, memoryToolResultText(result), tt.wantText)
		})
	}
}

func TestUpdateMemoryToolValidation(t *testing.T) {
	store, err := memory.NewFileStore(t.TempDir(), "MEMORIES.md", 128)
	require.NoError(t, err)
	tool := UpdateMemoryTool{Store: store}

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing operation", args: map[string]any{}},
		{name: "invalid operation", args: map[string]any{"operation": "delete"}},
		{name: "write without content", args: map[string]any{"operation": "write"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
		})
	}
}
