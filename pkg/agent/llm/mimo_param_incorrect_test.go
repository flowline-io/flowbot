package llm_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestMimoToolRoundTripPayload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		toolCount      int
		thinking       string
		wantAssistants int
	}{
		{
			name:           "one tool call with assistant text",
			toolCount:      1,
			thinking:       "need one file",
			wantAssistants: 1,
		},
		{
			name:           "twelve parallel tool calls",
			toolCount:      12,
			thinking:       "Need twelve independent lookups.",
			wantAssistants: 1,
		},
		{
			name:           "two parallel tool calls",
			toolCount:      2,
			thinking:       "two reads",
			wantAssistants: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parts := make([]msg.ContentPart, 0, tt.toolCount)
			results := make([]msg.AgentMessage, 0, tt.toolCount)
			reasoning := map[string]string{}
			for i := 0; i < tt.toolCount; i++ {
				id := fmt.Sprintf("call_%02d", i)
				args := fmt.Sprintf(`{"path":"f%d.go"}`, i)
				parts = append(parts, msg.ToolCallPart{ID: id, Name: "read_file", Arguments: args})
				results = append(results, msg.ToolResultMessage{
					ToolCallID: id,
					Name:       "read_file",
					Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
				})
				reasoning[id] = tt.thinking
			}
			messages := []msg.AgentMessage{
				msg.NewUserMessage("inspect the repo"),
				msg.AssistantMessage{ThinkingText: tt.thinking, Parts: parts},
			}
			messages = append(messages, results...)

			llmMessages, err := transform.DefaultConvertToLLM(messages)
			require.NoError(t, err)

			rec := &roundTripRecorder{responseBody: `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`}
			model, err := openai.New(
				openai.WithToken("test"),
				openai.WithModel("mimo-v2.5"),
				openai.WithBaseURL("https://api.example.com/v1"),
				openai.WithHTTPClient(llm.ThinkingHTTPClientForTest(rec)),
			)
			require.NoError(t, err)

			ctx := llm.WithThinkingLevel(context.Background(), llm.ThinkingLevelDefault)
			ctx = llm.WithAssistantToolReasoning(ctx, reasoning)
			_, err = model.GenerateContent(ctx, llmMessages, llms.WithModel("mimo-v2.5"))
			require.NoError(t, err)
			require.NotNil(t, rec.req)

			payload, err := io.ReadAll(rec.req.Body)
			require.NoError(t, err)
			var parsed map[string]any
			require.NoError(t, sonic.Unmarshal(payload, &parsed))
			rawMessages, ok := parsed["messages"].([]any)
			require.True(t, ok)

			var toolAssistantCount int
			for _, raw := range rawMessages {
				item, ok := raw.(map[string]any)
				require.True(t, ok)
				if item["role"] != "assistant" {
					continue
				}
				toolCalls, hasTools := item["tool_calls"].([]any)
				if !hasTools || len(toolCalls) == 0 {
					continue
				}
				toolAssistantCount++
				reasoningContent, hasReasoning := item["reasoning_content"].(string)
				assert.True(t, hasReasoning)
				assert.Equal(t, tt.thinking, reasoningContent)
				require.Len(t, toolCalls, tt.toolCount)
				for _, callRaw := range toolCalls {
					call, ok := callRaw.(map[string]any)
					require.True(t, ok)
					assert.NotEmpty(t, call["id"])
					assert.Equal(t, "function", call["type"])
					fn, ok := call["function"].(map[string]any)
					require.True(t, ok)
					assert.NotEmpty(t, fn["name"])
					args, ok := fn["arguments"].(string)
					require.True(t, ok)
					var obj map[string]any
					require.NoError(t, sonic.Unmarshal([]byte(args), &obj))
				}
			}
			assert.Equal(t, tt.wantAssistants, toolAssistantCount)
		})
	}
}
