package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/stretchr/testify/assert"
)

func TestSerializeConversation(t *testing.T) {
	tests := []struct {
		name string
		msgs []msg.AgentMessage
		want string
	}{
		{name: "user only", msgs: []msg.AgentMessage{msg.NewUserMessage("hi")}, want: "[User]: hi"},
		{name: "assistant with tool call", msgs: []msg.AgentMessage{
			msg.AssistantMessage{Parts: []msg.ContentPart{
				msg.TextPart{Text: "ok"},
				msg.ToolCallPart{Name: "read_file", Arguments: `{"path":"a.go"}`},
			}},
		}, want: "[Assistant tool calls]: read_file({\"path\":\"a.go\"})"},
		{name: "display only custom skipped", msgs: []msg.AgentMessage{
			msg.CustomMessage{DisplayOnly: true, Parts: []msg.ContentPart{msg.TextPart{Text: "hidden"}}},
		}, want: ""},
		{name: "tool result serialized", msgs: []msg.AgentMessage{
			msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "tool output"}}},
		}, want: "[Tool result]: tool output"},
		{name: "branch summary serialized", msgs: []msg.AgentMessage{
			msg.BranchSummaryMessage{Summary: "prior branch"},
		}, want: "[User]: prior branch"},
		{name: "compaction summary serialized", msgs: []msg.AgentMessage{
			msg.CompactionSummaryMessage{Summary: "compacted history"},
		}, want: "[User]: compacted history"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.SerializeConversation(tt.msgs)
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestExtractFileOpsFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  msg.AgentMessage
		wantRead string
		wantEdit string
	}{
		{name: "read", message: msg.AssistantMessage{Parts: []msg.ContentPart{
			msg.ToolCallPart{Name: "read_file", Arguments: `{"path":"main.go"}`},
		}}, wantRead: "main.go"},
		{name: "write", message: msg.AssistantMessage{Parts: []msg.ContentPart{
			msg.ToolCallPart{Name: "write_file", Arguments: `{"path":"out.txt"}`},
		}}},
		{name: "edit", message: msg.AssistantMessage{Parts: []msg.ContentPart{
			msg.ToolCallPart{Name: "edit_file", Arguments: `{"path":"pkg.go"}`},
		}}, wantEdit: "pkg.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ops := ctxmgr.NewFileOperations()
			ctxmgr.ExtractFileOpsFromMessage(tt.message, ops)
			if tt.wantRead != "" {
				_, ok := ops.Read[tt.wantRead]
				assert.True(t, ok)
			}
			if tt.wantEdit != "" {
				_, ok := ops.Edited[tt.wantEdit]
				assert.True(t, ok)
			}
		})
	}
}
