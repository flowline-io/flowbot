package chatagent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSubagentsForPrompt(t *testing.T) {
	tests := []struct {
		name       string
		subagents  []chatagent.Subagent
		wantParts  []string
		wantAbsent []string
	}{
		{
			name: "renders visible subagents",
			subagents: []chatagent.Subagent{{
				Name:        "code-reviewer",
				Description: "Reviews diffs for bugs and style",
			}},
			wantParts: []string{
				"<available_subagents>",
				"<name>code-reviewer</name>",
				"Reviews diffs for bugs and style",
				"delegate_subagent tool",
			},
		},
		{
			name: "skips subagents without description",
			subagents: []chatagent.Subagent{{
				Name:        "blank",
				Description: "   ",
			}},
			wantAbsent: []string{"<available_subagents>"},
		},
		{
			name:       "empty subagents",
			subagents:  nil,
			wantAbsent: []string{"<available_subagents>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatagent.FormatSubagentsForPrompt(tt.subagents)
			for _, part := range tt.wantParts {
				assert.Contains(t, got, part)
			}
			for _, part := range tt.wantAbsent {
				assert.NotContains(t, got, part)
			}
		})
	}
}

func TestTaskToolExecuteValidation(t *testing.T) {
	tool := chatagent.NewTaskTool(coding.Workspace{Root: t.TempDir()}, chatagent.TaskToolDeps{SessionID: "s1"})

	tests := []struct {
		name    string
		args    map[string]any
		wantErr string
	}{
		{
			name:    "missing subagent_type",
			args:    map[string]any{"prompt": "do something"},
			wantErr: "subagent_type is required",
		},
		{
			name:    "missing prompt",
			args:    map[string]any{"subagent_type": "reviewer"},
			wantErr: "prompt is required",
		},
		{
			name:    "blank values",
			args:    map[string]any{"subagent_type": "  ", "prompt": "  "},
			wantErr: "subagent_type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), "call-1", tt.args, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, toolResultText(result), tt.wantErr)
		})
	}
}

func TestTaskToolSchema(t *testing.T) {
	tool := chatagent.NewTaskTool(coding.Workspace{}, chatagent.TaskToolDeps{})
	assert.Equal(t, "delegate_subagent", tool.Name())
	assert.NotEmpty(t, tool.Description())
	params := tool.Parameters()
	required, ok := params["required"].([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"subagent_type", "description", "prompt"}, required)
}

func TestMigrateBuiltinSubagentFields(t *testing.T) {
	legacyGeneral := chatagent.LegacyBuiltinSystemPrompts["general"][0]
	legacyGeneralDesc := chatagent.LegacyBuiltinDescriptions["general"][0]
	legacyExplore := chatagent.LegacyBuiltinSystemPrompts["explore"][0]
	legacyExploreDesc := chatagent.LegacyBuiltinDescriptions["explore"][0]
	legacyScout := chatagent.LegacyBuiltinSystemPrompts["scout"][0]
	legacyScoutDesc := chatagent.LegacyBuiltinDescriptions["scout"][0]

	tests := []struct {
		name        string
		in          chatagent.BuiltinSubagentFields
		wantChanged bool
		wantPrompt  string
		wantDesc    string
	}{
		{
			name: "legacy builtin prompt and description migrate",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "general",
				SystemPrompt: legacyGeneral,
				Description:  legacyGeneralDesc,
			},
			wantChanged: true,
			wantPrompt:  "## Role",
			wantDesc:    legacyGeneralDesc,
		},
		{
			name: "customized builtin prompt is skipped",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "general",
				SystemPrompt: "Custom operator prompt",
				Description:  "Custom description",
			},
			wantChanged: false,
			wantPrompt:  "Custom operator prompt",
			wantDesc:    "Custom description",
		},
		{
			name: "non-builtin source is skipped",
			in: chatagent.BuiltinSubagentFields{
				Source:       "user",
				Flag:         "general",
				SystemPrompt: legacyGeneral,
				Description:  legacyGeneralDesc,
			},
			wantChanged: false,
			wantPrompt:  legacyGeneral,
			wantDesc:    legacyGeneralDesc,
		},
		{
			name: "unknown builtin flag is skipped",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "custom-flag",
				SystemPrompt: legacyGeneral,
				Description:  legacyGeneralDesc,
			},
			wantChanged: false,
			wantPrompt:  legacyGeneral,
			wantDesc:    legacyGeneralDesc,
		},
		{
			name: "already migrated prompt is skipped",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "general",
				SystemPrompt: "already-new-prompt",
				Description:  legacyGeneralDesc,
			},
			wantChanged: false,
			wantPrompt:  "already-new-prompt",
			wantDesc:    legacyGeneralDesc,
		},
		{
			name: "legacy explore prompt migrates",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "explore",
				SystemPrompt: legacyExplore,
				Description:  legacyExploreDesc,
			},
			wantChanged: true,
			wantPrompt:  "## Role",
			wantDesc:    legacyExploreDesc,
		},
		{
			name: "legacy scout prompt migrates",
			in: chatagent.BuiltinSubagentFields{
				Source:       "builtin",
				Flag:         "scout",
				SystemPrompt: legacyScout,
				Description:  legacyScoutDesc,
			},
			wantChanged: true,
			wantPrompt:  "## Role",
			wantDesc:    legacyScoutDesc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := chatagent.MigrateBuiltinSubagentFields(tt.in)
			assert.Equal(t, tt.wantChanged, changed)
			if tt.wantChanged {
				assert.Contains(t, got.SystemPrompt, tt.wantPrompt)
				assert.Equal(t, tt.wantDesc, got.Description)
				again, againChanged := chatagent.MigrateBuiltinSubagentFields(got)
				assert.False(t, againChanged)
				assert.Equal(t, got, again)
				return
			}
			assert.Equal(t, tt.wantPrompt, got.SystemPrompt)
			assert.Equal(t, tt.wantDesc, got.Description)
		})
	}
}

func toolResultText(result msg.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			if _, err := out.WriteString(tp.Text); err != nil {
				return out.String()
			}
		}
	}
	return out.String()
}
