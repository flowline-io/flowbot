package transform_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestDefaultConvertToLLM(t *testing.T) {
	tests := []struct {
		name     string
		messages []agent.AgentMessage
		wantLen  int
		wantRole llms.ChatMessageType
	}{
		{
			name:     "user message",
			messages: []agent.AgentMessage{agent.NewUserMessage("hello")},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeHuman,
		},
		{
			name: "assistant with tool call",
			messages: []agent.AgentMessage{agent.AssistantMessage{Parts: []agent.ContentPart{
				agent.TextPart{Text: "thinking"},
				agent.ToolCallPart{ID: "1", Name: "echo", Arguments: `{}`},
			}}},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeAI,
		},
		{
			name: "custom display only filtered",
			messages: []agent.AgentMessage{
				agent.CustomMessage{DisplayOnly: true, Parts: []agent.ContentPart{agent.TextPart{Text: "hidden"}}},
				agent.NewUserMessage("visible"),
			},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeHuman,
		},
		{
			name:     "branch summary",
			messages: []agent.AgentMessage{agent.BranchSummaryMessage{Summary: "branch context"}},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeHuman,
		},
		{
			name:     "compaction summary",
			messages: []agent.AgentMessage{agent.CompactionSummaryMessage{Summary: "compact context"}},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeHuman,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := transform.DefaultConvertToLLM(tt.messages)
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantRole, result[0].Role)
			}
		})
	}
}

func TestProcessAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []transform.Attachment
		wantParts   int
	}{
		{name: "url attachment", attachments: []transform.Attachment{{URL: "http://img", MIMEType: "image/png"}}, wantParts: 1},
		{name: "binary attachment", attachments: []transform.Attachment{{MIMEType: "image/jpeg", Data: []byte("abc")}}, wantParts: 1},
		{name: "empty list", attachments: nil, wantParts: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parts := transform.ProcessAttachments(tt.attachments)
			assert.Len(t, parts, tt.wantParts)
		})
	}
}

func TestMergeSystemPrompt(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		extra string
		want  string
	}{
		{name: "both present", base: "base", extra: "extra", want: "base\n\nextra"},
		{name: "only base", base: "base", extra: "", want: "base"},
		{name: "only extra", base: "", extra: "extra", want: "extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, transform.MergeSystemPrompt(tt.base, tt.extra))
		})
	}
}
