package session_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalEntryDurationFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
	}{
		{
			name: "assistant timing fields",
			line: `{"id":"1","type":"message","message":{"role":"assistant","text":"hi","turn_duration_ms":1200,"thinking_duration_ms":300,"thinking_text":"plan","run_duration_ms":5000}}`,
		},
		{
			name: "tool result duration",
			line: `{"id":"2","type":"message","message":{"role":"toolResult","tool_call_id":"c1","name":"echo","text":"ok","duration_ms":42}}`,
		},
		{
			name: "legacy without duration",
			line: `{"id":"3","type":"message","message":{"role":"assistant","text":"legacy"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry, err := session.UnmarshalEntry([]byte(tt.line))
			require.NoError(t, err)
			roundTrip, err := session.MarshalEntry(entry)
			require.NoError(t, err)
			entry2, err := session.UnmarshalEntry(roundTrip)
			require.NoError(t, err)

			switch message := entry2.Message.(type) {
			case msg.AssistantMessage:
				if tt.name == "assistant timing fields" {
					assert.Equal(t, int64(1200), message.TurnDurationMs)
					assert.Equal(t, int64(300), message.ThinkingDurationMs)
					assert.Equal(t, "plan", message.ThinkingText)
					assert.Equal(t, int64(5000), message.RunDurationMs)
				} else {
					assert.Equal(t, int64(0), message.RunDurationMs)
				}
			case msg.ToolResultMessage:
				assert.Equal(t, int64(42), message.DurationMs)
			default:
				t.Fatalf("unexpected message type %T", entry2.Message)
			}
		})
	}
}
