package chatagent

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestHistoryMessagesFromMessage(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		in   msg.AgentMessage
		want int
		kind string
		ms   int64
	}{
		{
			name: "tool result row",
			in: msg.ToolResultMessage{
				Name:       "echo",
				Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
				DurationMs: 88,
			},
			want: 1,
			kind: "tool",
			ms:   88,
		},
		{
			name: "thinking and assistant rows",
			in: msg.AssistantMessage{
				Parts:              []msg.ContentPart{msg.TextPart{Text: "answer"}},
				ThinkingText:       "plan",
				ThinkingDurationMs: 200,
				TurnDurationMs:     900,
				RunDurationMs:      4000,
			},
			want: 2,
			kind: "thinking",
			ms:   200,
		},
		{
			name: "user row",
			in:   msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
			want: 1,
			kind: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := historyMessagesFromMessage(tt.in, createdAt)
			assert.Len(t, got, tt.want)
			if tt.want == 0 {
				return
			}
			assert.Equal(t, tt.kind, got[0].Kind)
			if tt.kind == "tool" {
				assert.Equal(t, tt.ms, got[0].DurationMs)
			}
			if tt.kind == "thinking" {
				assert.Equal(t, tt.ms, got[0].ThinkingDurationMs)
				assert.Equal(t, "assistant", got[1].Kind)
				assert.Equal(t, int64(900), got[1].TurnDurationMs)
				assert.Equal(t, int64(4000), got[1].RunDurationMs)
			}
		})
	}
}

func TestHasPersistedToolResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []HistoryMessage
		want bool
	}{
		{name: "has tool", in: []HistoryMessage{{Kind: "tool"}}, want: true},
		{name: "assistant only", in: []HistoryMessage{{Kind: "assistant"}}, want: false},
		{name: "empty", in: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasPersistedToolResults(tt.in))
		})
	}
}
