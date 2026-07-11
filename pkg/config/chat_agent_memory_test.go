package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDirectory(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })

	root := t.TempDir()
	workspace := filepath.Join(root, "chat-workspace")
	require.NoError(t, os.MkdirAll(workspace, 0o755))

	tests := []struct {
		name      string
		workspace string
		wantErr   bool
		wantSub   string
	}{
		{
			name:      "sibling agent-memories directory",
			workspace: workspace,
			wantSub:   "agent-memories",
		},
		{
			name:      "missing workspace",
			workspace: "",
			wantErr:   true,
		},
		{
			name:      "whitespace workspace",
			workspace: "   ",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.Workspace = tt.workspace
			got, err := config.MemoryDirectory()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, got, tt.wantSub)
		})
	}
}

func TestChatAgentDefaultMemoryFileAndMaxBytes(t *testing.T) {
	tests := []struct {
		name     string
		wantFile string
		wantMax  int
	}{
		{name: "default file", wantFile: "MEMORIES.md", wantMax: 65536},
		{name: "stable defaults", wantFile: "MEMORIES.md", wantMax: 65536},
		{name: "max bytes constant", wantFile: "MEMORIES.md", wantMax: 65536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantFile, config.ChatAgentDefaultMemoryFile())
			assert.Equal(t, tt.wantMax, config.ChatAgentMemoryMaxFileBytes())
		})
	}
}
