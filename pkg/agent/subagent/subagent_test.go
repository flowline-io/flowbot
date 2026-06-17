package subagent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/subagent"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		deps         func() subagent.Deps
		wantErr      error
		wantText     string
		wantProgress bool
	}{
		{
			name: "happy path returns final assistant text",
			deps: func() subagent.Deps {
				return subagent.Deps{
					Model:    llm.NewFakeModel(llm.ResponseScript{Content: "final answer"}),
					Registry: tool.NewRegistry(),
					MaxDepth: 1,
				}
			},
			wantText: "final answer",
		},
		{
			name: "tool call forwards progress and returns text",
			deps: func() subagent.Deps {
				registry := tool.NewRegistry()
				require.NoError(t, registry.Register(echo.Tool{}))
				return subagent.Deps{
					Model: llm.NewFakeModel(
						llm.ResponseScript{ToolCalls: []llms.ToolCall{{
							ID:   "call-1",
							Type: "function",
							FunctionCall: &llms.FunctionCall{
								Name:      "echo",
								Arguments: `{"text":"hi"}`,
							},
						}}},
						llm.ResponseScript{Content: "done with tool"},
					),
					Registry: registry,
					MaxDepth: 1,
				}
			},
			wantText:     "done with tool",
			wantProgress: true,
		},
		{
			name: "max depth exceeded",
			deps: func() subagent.Deps {
				return subagent.Deps{
					Model:    llm.NewFakeModel(llm.ResponseScript{Content: "x"}),
					Registry: tool.NewRegistry(),
					Depth:    1,
					MaxDepth: 1,
				}
			},
			wantErr: subagent.ErrMaxDepth,
		},
		{
			name: "missing model",
			deps: func() subagent.Deps {
				return subagent.Deps{Registry: tool.NewRegistry(), MaxDepth: 1}
			},
			wantErr: subagent.ErrNoModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var progress []string
			def := subagent.Definition{
				Name:         "tester",
				Description:  "test subagent",
				SystemPrompt: "You are a test subagent.",
			}
			result, err := subagent.Run(context.Background(), def, tt.deps(), "do the task", func(update string) {
				progress = append(progress, update)
			})
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, result.Text)
			if tt.wantProgress {
				assert.NotEmpty(t, progress)
				assert.True(t, containsSubstr(progress, "echo"), "expected progress to mention tool, got %v", progress)
			}
		})
	}
}

func containsSubstr(items []string, substr string) bool {
	for _, item := range items {
		if strings.Contains(item, substr) {
			return true
		}
	}
	return false
}
