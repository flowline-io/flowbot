package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectableSubagentToolsIncludesMemory(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })

	tests := []struct {
		name string
	}{
		{name: "includes update_memory in selectable tools"},
		{name: "count matches coding tools plus memory"},
		{name: "memory name is update_memory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := SelectableSubagentTools()
			assert.Len(t, names, len(coding.ActiveToolNames())+1)
			assert.Contains(t, names, updateMemoryToolName)
		})
	}
}

func TestActiveToolNamesIncludesMemory(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "interactive default includes memory"},
		{name: "memory appears once"},
		{name: "memory is update_memory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := ActiveToolNames()
			assert.Contains(t, names, updateMemoryToolName)
		})
	}
}

func TestBaseToolNamesForRun(t *testing.T) {
	tests := []struct {
		name          string
		kind          RunKind
		explicitTools []string
		wantMemory    bool
	}{
		{name: "interactive default includes memory", kind: RunKindInteractive, wantMemory: true},
		{name: "pipeline default omits memory", kind: RunKindPipeline, wantMemory: false},
		{name: "scheduled default omits memory", kind: RunKindScheduled, wantMemory: false},
		{name: "pipeline explicit allowlist keeps memory", kind: RunKindPipeline, explicitTools: []string{"read_file", updateMemoryToolName}, wantMemory: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := BaseToolNamesForRun(tt.kind, tt.explicitTools)
			if tt.wantMemory {
				assert.Contains(t, names, updateMemoryToolName)
				return
			}
			assert.NotContains(t, names, updateMemoryToolName)
		})
	}
}

func TestNewRegistryRegistersMemory(t *testing.T) {
	LockAppConfigForTest(t)
	root := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{
		ChatModel: "gpt-test",
		Workspace: root,
	}

	ws, err := WorkspaceFromConfig()
	require.NoError(t, err)
	reg, err := NewRegistry(ws, nil, nil)
	require.NoError(t, err)
	_, ok := reg.Get(updateMemoryToolName)
	assert.True(t, ok)
	_, ok = reg.Get(searchKnowledgeToolName)
	assert.True(t, ok)
	_, ok = reg.Get(getKnowledgeToolName)
	assert.True(t, ok)
}

func TestNewSubagentRegistryRegistersMemory(t *testing.T) {
	LockAppConfigForTest(t)
	root := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{
		ChatModel: "gpt-test",
		Workspace: root,
	}

	ws, err := WorkspaceFromConfig()
	require.NoError(t, err)
	reg, err := NewSubagentRegistry(ws, nil)
	require.NoError(t, err)
	_, ok := reg.Get(updateMemoryToolName)
	assert.True(t, ok)
}
