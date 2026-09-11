package chatagent

import (
	"testing"

	agenthtml "github.com/flowline-io/flowbot/internal/server/chatagent/tools/htmlpreview"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
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
			assert.NotContains(t, names, agenthtml.ToolName)
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
		name            string
		kind            RunKind
		explicitTools   []string
		wantMemory      bool
		wantPresentHTML bool
	}{
		{name: "interactive default includes memory", kind: RunKindInteractive, wantMemory: true, wantPresentHTML: true},
		{name: "pipeline default omits memory", kind: RunKindPipeline, wantMemory: false, wantPresentHTML: false},
		{name: "scheduled default omits memory", kind: RunKindScheduled, wantMemory: false, wantPresentHTML: true},
		{name: "pipeline explicit allowlist keeps memory", kind: RunKindPipeline, explicitTools: []string{"read_file", memorySetToolName}, wantMemory: true, wantPresentHTML: false},
		{name: "pipeline explicit list keeps present_html", kind: RunKindPipeline, explicitTools: []string{agenthtml.ToolName}, wantPresentHTML: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := BaseToolNamesForRun(tt.kind, tt.explicitTools)
			if tt.wantPresentHTML {
				assert.Contains(t, names, agenthtml.ToolName)
			} else {
				assert.NotContains(t, names, agenthtml.ToolName)
			}
			if tt.wantMemory {
				assert.Contains(t, names, memorySetToolName)
				return
			}
			if len(tt.explicitTools) > 0 {
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

func TestNewSubagentRegistryOmitsPresentHTML(t *testing.T) {
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
	_, ok := reg.Get(agenthtml.ToolName)
	assert.False(t, ok)
}
