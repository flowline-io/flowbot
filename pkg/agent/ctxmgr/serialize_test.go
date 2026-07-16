package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/stretchr/testify/assert"
)

func TestSerializeConversation(t *testing.T) {
	tests := []struct {
		name string
		msgs []agent.AgentMessage
		want string
	}{
		{name: "user only", msgs: []agent.AgentMessage{agent.NewUserMessage("hi")}, want: "[User]: hi"},
		{name: "assistant with tool call", msgs: []agent.AgentMessage{
			agent.AssistantMessage{Parts: []agent.ContentPart{
				agent.TextPart{Text: "ok"},
				agent.ToolCallPart{Name: "read_file", Arguments: `{"path":"a.go"}`},
			}},
		}, want: "[Assistant tool calls]: read_file({\"path\":\"a.go\"})"},
		{name: "display only custom skipped", msgs: []agent.AgentMessage{
			agent.CustomMessage{DisplayOnly: true, Parts: []agent.ContentPart{agent.TextPart{Text: "hidden"}}},
		}, want: ""},
		{name: "tool result serialized", msgs: []agent.AgentMessage{
			agent.ToolResultMessage{Parts: []agent.ContentPart{agent.TextPart{Text: "tool output"}}},
		}, want: "[Tool result]: tool output"},
		{name: "branch summary serialized", msgs: []agent.AgentMessage{
			agent.BranchSummaryMessage{Summary: "prior branch"},
		}, want: "[User]: prior branch"},
		{name: "compaction summary serialized", msgs: []agent.AgentMessage{
			agent.CompactionSummaryMessage{Summary: "compacted history"},
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
		message  agent.AgentMessage
		wantRead string
		wantEdit string
	}{
		{name: "read", message: agent.AssistantMessage{Parts: []agent.ContentPart{
			agent.ToolCallPart{Name: "read_file", Arguments: `{"path":"main.go"}`},
		}}, wantRead: "main.go"},
		{name: "write", message: agent.AssistantMessage{Parts: []agent.ContentPart{
			agent.ToolCallPart{Name: "write_file", Arguments: `{"path":"out.txt"}`},
		}}},
		{name: "edit", message: agent.AssistantMessage{Parts: []agent.ContentPart{
			agent.ToolCallPart{Name: "edit_file", Arguments: `{"path":"pkg.go"}`},
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
