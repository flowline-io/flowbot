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
		{name: "includes memory_set in selectable tools"},
		{name: "count matches coding tools plus memory tools"},
		{name: "includes search_session_summaries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := SelectableSubagentTools()
			assert.Len(t, names, len(coding.ActiveToolNames())+len(MemoryToolNames()))
			assert.Contains(t, names, memorySetToolName)
			assert.Contains(t, names, searchSessionSummariesToolName)
		})
	}
}

func TestActiveToolNamesIncludesMemory(t *testing.T) {
	tests := []struct {
		name string
		tool string
	}{
		{name: "includes memory_set", tool: memorySetToolName},
		{name: "includes memory_get", tool: memoryGetToolName},
		{name: "includes search_session_summaries", tool: searchSessionSummariesToolName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, ActiveToolNames(), tt.tool)
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
		{name: "pipeline explicit allowlist keeps memory", kind: RunKindPipeline, explicitTools: []string{"read_file", memorySetToolName}, wantMemory: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := BaseToolNamesForRun(tt.kind, tt.explicitTools)
			if tt.wantMemory {
				assert.Contains(t, names, memorySetToolName)
				return
			}
			assert.NotContains(t, names, memorySetToolName)
			assert.NotContains(t, names, searchSessionSummariesToolName)
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
	for _, name := range MemoryToolNames() {
		_, ok := reg.Get(name)
		assert.True(t, ok, "missing tool %s", name)
	}
	_, ok := reg.Get(searchKnowledgeToolName)
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
	_, ok := reg.Get(memorySetToolName)
	assert.True(t, ok)
}
