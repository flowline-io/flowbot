package ctxmgr_test

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestPruneToolOutputs(t *testing.T) {
	toolResult := func(name, text string) msg.ToolResultMessage {
		return msg.ToolResultMessage{
			Name:  name,
			Parts: []msg.ContentPart{msg.TextPart{Text: text}},
		}
	}

	tests := []struct {
		name     string
		settings ctxmgr.Settings
		messages []msg.AgentMessage
		wantLen  int
	}{
		{
			name:     "keeps messages when prune disabled",
			settings: ctxmgr.Settings{PruneToolOutputs: false},
			messages: []msg.AgentMessage{agent.NewUserMessage("hi"), toolResult("read", strings.Repeat("a", 120000))},
			wantLen:  2,
		},
		{
			name:     "keeps small tool output batches",
			settings: ctxmgr.Settings{PruneToolOutputs: true},
			messages: []msg.AgentMessage{agent.NewUserMessage("hi"), toolResult("read", strings.Repeat("a", 10000))},
			wantLen:  2,
		},
		{
			name:     "prunes old large tool outputs but keeps recent messages",
			settings: ctxmgr.Settings{PruneToolOutputs: true},
			messages: []msg.AgentMessage{
				agent.NewUserMessage("first"),
				toolResult("read", strings.Repeat("a", 120000)),
				agent.NewUserMessage("second"),
				toolResult("read", strings.Repeat("b", 120000)),
				agent.NewUserMessage("recent"),
			},
			wantLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.PruneToolOutputs(tt.messages, tt.settings)
			assert.Len(t, got, tt.wantLen)
			assert.Equal(t, tt.messages[0].Role(), got[0].Role())
			assert.Equal(t, tt.messages[len(tt.messages)-1].Role(), got[len(got)-1].Role())
			if tt.name == "prunes old large tool outputs but keeps recent messages" {
				toolCount := 0
				for _, message := range got {
					if _, ok := message.(msg.ToolResultMessage); ok {
						toolCount++
					}
				}
				assert.Equal(t, 1, toolCount)
			}
		})
	}
}
