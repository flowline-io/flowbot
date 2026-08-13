package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestExtractFileOpsFromMessage(t *testing.T) {
	t.Parallel()
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
