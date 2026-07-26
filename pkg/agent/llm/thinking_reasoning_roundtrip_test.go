package llm_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThinkingHTTPClientInjectsReasoningContentForToolCalls(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		level         string
		reasoning     map[string]string
		body          string
		wantReasoning []string // per assistant+tool_calls message; empty slice means field absent
	}{
		{
			name: "deepseek injects prior tool-call reasoning by id",
			body: `{
				"model":"deepseek-v4-flash",
				"messages":[
					{"role":"user","content":"hi"},
					{"role":"assistant","content":"","tool_calls":[{"id":"call_a","type":"function","function":{"name":"read_skill","arguments":"{}"}}]},
					{"role":"tool","tool_call_id":"call_a","content":"ok"}
				]
			}`,
			reasoning:     map[string]string{"call_a": "need gitea skill"},
			wantReasoning: []string{"need gitea skill"},
		},
		{
			name: "mimo injects empty reasoning when missing",
			body: `{
				"model":"mimo-v2.5",
				"messages":[
					{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{}"}}]}
				]
			}`,
			reasoning:     nil,
			wantReasoning: []string{""},
		},
		{
			name: "preserves existing reasoning_content",
			body: `{
				"model":"deepseek-v4-flash",
				"messages":[
					{"role":"assistant","reasoning_content":"kept","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{}"}}]}
				]
			}`,
			reasoning:     map[string]string{"1": "ignored"},
			wantReasoning: []string{"kept"},
		},
		{
			name:  "thinking off still injects for tool-call history",
			level: llm.ThinkingLevelOff,
			body: `{
				"model":"deepseek-v4-flash",
				"messages":[
					{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{}"}}]}
				]
			}`,
			reasoning:     map[string]string{"1": "prior think"},
			wantReasoning: []string{"prior think"},
		},
		{
			name: "skips assistant without tool_calls",
			body: `{
				"model":"deepseek-v4-flash",
				"messages":[
					{"role":"assistant","content":"plain answer"},
					{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{}"}}]}
				]
			}`,
			reasoning:     map[string]string{"1": "only for tools"},
			wantReasoning: []string{"only for tools"},
		},
		{
			name: "matches by tool call id when order differs from collect order",
			body: `{
				"model":"deepseek-v4-flash",
				"messages":[
					{"role":"assistant","tool_calls":[{"id":"second","type":"function","function":{"name":"echo","arguments":"{}"}}]},
					{"role":"assistant","tool_calls":[{"id":"first","type":"function","function":{"name":"echo","arguments":"{}"}}]}
				]
			}`,
			reasoning:     map[string]string{"first": "A", "second": "B"},
			wantReasoning: []string{"B", "A"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &roundTripRecorder{}
			client := llm.ThinkingHTTPClientForTest(rec)
			req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/chat/completions", strings.NewReader(tt.body))
			require.NoError(t, err)
			ctx := llm.WithThinkingLevel(req.Context(), tt.level)
			ctx = llm.WithAssistantToolReasoning(ctx, tt.reasoning)
			req = req.WithContext(ctx)

			_, err = client.Do(req)
			require.NoError(t, err)

			payload, err := io.ReadAll(rec.req.Body)
			require.NoError(t, err)
			var parsed map[string]any
			require.NoError(t, sonic.Unmarshal(payload, &parsed))
			messages, ok := parsed["messages"].([]any)
			require.True(t, ok)

			got := make([]string, 0, 2)
			for _, raw := range messages {
				msg, ok := raw.(map[string]any)
				require.True(t, ok)
				if msg["role"] != "assistant" {
					continue
				}
				toolCalls, hasTools := msg["tool_calls"].([]any)
				if !hasTools || len(toolCalls) == 0 {
					_, hasReasoning := msg["reasoning_content"]
					assert.False(t, hasReasoning, "plain assistant must not get reasoning_content")
					continue
				}
				if reasoning, has := msg["reasoning_content"].(string); has {
					got = append(got, reasoning)
				}
			}
			if len(tt.wantReasoning) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.wantReasoning, got)
		})
	}
}
