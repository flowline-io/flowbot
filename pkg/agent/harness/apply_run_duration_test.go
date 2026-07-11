package harness

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestApplyRunDuration(t *testing.T) {
	tests := []struct {
		name      string
		messages  []any
		runStart  time.Time
		wantRunMs int64
		wantIdx   int
	}{
		{
			name: "sets run duration on final assistant with text",
			messages: []any{
				agent.NewUserMessage("hi"),
				msg.AssistantMessage{Parts: []msg.ContentPart{msg.ToolCallPart{Name: "echo"}}},
				msg.ToolResultMessage{Name: "echo", Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}}},
				msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "done"}}},
			},
			runStart:  time.Now().Add(-2 * time.Second),
			wantRunMs: 2000,
			wantIdx:   3,
		},
		{
			name: "falls back to last assistant without text",
			messages: []any{
				msg.AssistantMessage{Parts: []msg.ContentPart{msg.ToolCallPart{Name: "echo"}}},
			},
			runStart:  time.Now().Add(-1500 * time.Millisecond),
			wantRunMs: 1500,
			wantIdx:   0,
		},
		{
			name: "zero run start leaves messages unchanged",
			messages: []any{
				msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}},
			},
			runStart:  time.Time{},
			wantRunMs: 0,
			wantIdx:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := applyRunDuration(tt.messages, tt.runStart)
			assistant, ok := got[tt.wantIdx].(msg.AssistantMessage)
			assert.True(t, ok)
			if tt.wantRunMs == 0 {
				assert.Equal(t, int64(0), assistant.RunDurationMs)
				return
			}
			assert.GreaterOrEqual(t, assistant.RunDurationMs, tt.wantRunMs-50)
			assert.LessOrEqual(t, assistant.RunDurationMs, tt.wantRunMs+200)
		})
	}
}
