package llm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
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

func TestStreamAssistant_FiltersToolCallStreamDeltas(t *testing.T) {
	tests := []struct {
		name        string
		chunks      []string
		toolCalls   []llms.ToolCall
		wantText    string
		wantContent string
	}{
		{
			name: "drops langchaingo tool call stream chunks",
			chunks: []string{
				`[{"id":"call_00","type":"function","function":{"name":"write_file","arguments":""}}]`,
				`[{"type":"","function":{"name":"","arguments":"{\"path\""}}]`,
				`[{"type":"","function":{"name":"","arguments":": \"x.py\"}"}}]`,
			},
			toolCalls: []llms.ToolCall{{
				ID:   "call_00",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "write_file",
					Arguments: `{"path": "x.py"}`,
				},
			}},
			wantText:    "",
			wantContent: "",
		},
		{
			name: "keeps visible text before tool call chunks",
			chunks: []string{
				"I will write a file.",
				`[{"id":"call_00","type":"function","function":{"name":"write_file","arguments":""}}]`,
			},
			toolCalls: []llms.ToolCall{{
				ID:           "call_00",
				Type:         "function",
				FunctionCall: &llms.FunctionCall{Name: "write_file", Arguments: `{}`},
			}},
			wantText:    "I will write a file.",
			wantContent: "I will write a file.",
		},
		{
			name: "plain text stream unchanged",
			chunks: []string{
				"hello",
				" world",
			},
			wantText:    "hello world",
			wantContent: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := llm.NewFakeModel(llm.ResponseScript{
				Chunks:    tt.chunks,
				ToolCalls: tt.toolCalls,
			})
			var text strings.Builder

			result, err := llm.StreamAssistant(context.Background(), model, "", nil, llm.StreamOptions{
				ModelName: "fake",
				OnTextDelta: func(delta string) error {
					text.WriteString(delta)
					return nil
				},
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantText, text.String())
			assert.Equal(t, tt.wantContent, result.Content)
			if len(tt.toolCalls) > 0 {
				require.Len(t, result.ToolCalls, 1)
				assert.Equal(t, tt.toolCalls[0].FunctionCall.Name, result.ToolCalls[0].FunctionCall.Name)
			}
		})
	}
}
