package ctxmgr_test

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneToolOutputs(t *testing.T) {
	t.Parallel()
	toolResult := func(id, name, text string) msg.ToolResultMessage {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       name,
			Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		}
	}
	oversized := strings.Repeat("a", 20000)
	alreadyPruned := strings.Repeat("h", 4096) + "\n\n[... tool result middle pruned ...]\n\n" + strings.Repeat("t", 1024)

	tests := []struct {
		name          string
		settings      ctxmgr.Settings
		messages      []msg.AgentMessage
		wantLen       int
		wantMarker    bool
		wantUnchanged bool
	}{
		{
			name:          "keeps messages when prune disabled",
			settings:      ctxmgr.Settings{PruneToolOutputs: false},
			messages:      []msg.AgentMessage{msg.NewUserMessage("hi"), toolResult("c1", "read", oversized)},
			wantLen:       2,
			wantUnchanged: true,
		},
		{
			name:          "keeps small tool output batches",
			settings:      ctxmgr.Settings{PruneToolOutputs: true},
			messages:      []msg.AgentMessage{msg.NewUserMessage("hi"), toolResult("c1", "read", strings.Repeat("a", 1000))},
			wantLen:       2,
			wantUnchanged: true,
		},
		{
			name:     "rewrites oversized tool output to head marker tail",
			settings: ctxmgr.Settings{PruneToolOutputs: true},
			messages: []msg.AgentMessage{
				msg.NewUserMessage("first"),
				toolResult("c1", "read", oversized),
				msg.NewUserMessage("recent"),
			},
			wantLen:    3,
			wantMarker: true,
		},
		{
			name:          "skips already pruned tool output under budget",
			settings:      ctxmgr.Settings{PruneToolOutputs: true},
			messages:      []msg.AgentMessage{toolResult("c1", "read", alreadyPruned)},
			wantLen:       1,
			wantUnchanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.PruneToolOutputs(tt.messages, tt.settings)
			require.Len(t, got, tt.wantLen)
			assert.Equal(t, tt.messages[0].Role(), got[0].Role())
			if tt.wantUnchanged {
				assert.Equal(t, tt.messages, got)
				return
			}
			if !tt.wantMarker {
				return
			}
			tool, ok := got[1].(msg.ToolResultMessage)
			require.True(t, ok)
			var text strings.Builder
			for _, part := range tool.Parts {
				if tp, ok := part.(msg.TextPart); ok {
					text.WriteString(tp.Text)
				}
			}
			assert.Contains(t, text.String(), "[... tool result middle pruned ...]")
			assert.Less(t, len([]rune(text.String())), len([]rune(oversized)))
			assert.True(t, strings.HasPrefix(text.String(), strings.Repeat("a", 32)))
			assert.True(t, strings.HasSuffix(text.String(), strings.Repeat("a", 32)))
			assert.Equal(t, "c1", tool.ToolCallID)
		})
	}
}
