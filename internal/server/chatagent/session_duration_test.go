package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
)

func TestSumRunDurationFromBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		branch []session.TreeEntry
		want   int64
	}{
		{
			name: "sums assistant run durations",
			branch: []session.TreeEntry{
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts:         []msg.ContentPart{msg.TextPart{Text: "first"}},
						RunDurationMs: 1200,
					},
				},
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts:         []msg.ContentPart{msg.TextPart{Text: "second"}},
						RunDurationMs: 3400,
					},
				},
			},
			want: 4600,
		},
		{
			name: "ignores user and assistant without run duration",
			branch: []session.TreeEntry{
				{
					Type:    session.EntryMessage,
					Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}},
				},
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts: []msg.ContentPart{msg.TextPart{Text: "reply"}},
					},
				},
				{
					Type: session.EntryMessage,
					Message: msg.ToolResultMessage{
						Name:       "echo",
						Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
						DurationMs: 50,
					},
				},
			},
			want: 0,
		},
		{
			name:   "empty branch",
			branch: nil,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, sumRunDurationFromBranch(tt.branch))
		})
	}
}
