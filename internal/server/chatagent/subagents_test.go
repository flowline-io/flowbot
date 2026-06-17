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
				"task tool",
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
	assert.Equal(t, "task", tool.Name())
	assert.NotEmpty(t, tool.Description())
	params := tool.Parameters()
	required, ok := params["required"].([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"subagent_type", "description", "prompt"}, required)
}

func toolResultText(result msg.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			out.WriteString(tp.Text)
		}
	}
	return out.String()
}
