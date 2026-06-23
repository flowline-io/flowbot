package llm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamAssistant_ReasoningDelta(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		script        llm.ResponseScript
		wantReasoning string
		wantText      string
		wantContent   string
	}{
		{
			name:      "deepseek v4 streams reasoning content field",
			modelName: "deepseek-v4-flash",
			script: llm.ResponseScript{
				ReasoningChunks: []string{"step"},
				Chunks:          []string{"ok"},
			},
			wantReasoning: "step",
			wantText:      "ok",
			wantContent:   "ok",
		},
		{
			name:      "streams reasoning and answer separately",
			modelName: "claude-sonnet-4",
			script: llm.ResponseScript{
				ReasoningChunks: []string{"think", "ing"},
				Chunks:          []string{"ans", "wer"},
			},
			wantReasoning: "thinking",
			wantText:      "answer",
			wantContent:   "answer",
		},
		{
			name:      "reasoning only leaves answer empty",
			modelName: "gpt-5.3-codex",
			script: llm.ResponseScript{
				ReasoningChunks: []string{"plan"},
			},
			wantReasoning: "plan",
			wantText:      "",
			wantContent:   "",
		},
		{
			name:      "answer without reasoning still streams text",
			modelName: "claude-opus-4",
			script: llm.ResponseScript{
				Chunks: []string{"hi"},
			},
			wantReasoning: "",
			wantText:      "hi",
			wantContent:   "hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := llm.NewFakeModel(tt.script)
			var reasoning strings.Builder
			var text strings.Builder

			result, err := llm.StreamAssistant(context.Background(), model, "", nil, llm.StreamOptions{
				ModelName: tt.modelName,
				OnReasoningDelta: func(delta string) error {
					reasoning.WriteString(delta)
					return nil
				},
				OnTextDelta: func(delta string) error {
					text.WriteString(delta)
					return nil
				},
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantReasoning, reasoning.String())
			assert.Equal(t, tt.wantText, text.String())
			assert.Equal(t, tt.wantContent, result.Content)
		})
	}
}
