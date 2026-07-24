package transform_test

import (
	"strings"
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
		check    func(t *testing.T, result []llms.MessageContent)
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
			name: "assistant with empty tool call id gets synthesized",
			messages: []agent.AgentMessage{agent.AssistantMessage{Parts: []agent.ContentPart{
				agent.ToolCallPart{ID: "", Name: "echo", Arguments: `{}`},
			}}},
			wantLen:  1,
			wantRole: llms.ChatMessageTypeAI,
			check: func(t *testing.T, result []llms.MessageContent) {
				require.Len(t, result[0].Parts, 1)
				tc, ok := result[0].Parts[0].(llms.ToolCall)
				require.True(t, ok)
				assert.NotEmpty(t, tc.ID)
				assert.True(t, strings.HasPrefix(tc.ID, "call_"))
			},
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
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestProcessAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []transform.Attachment
		wantParts   int
		wantKind    agent.MediaKind
	}{
		{name: "url attachment", attachments: []transform.Attachment{{URL: "http://img", MIMEType: "image/png"}}, wantParts: 1, wantKind: agent.MediaKindImage},
		{name: "binary attachment", attachments: []transform.Attachment{{MIMEType: "image/jpeg", Data: []byte("abc")}}, wantParts: 1, wantKind: agent.MediaKindImage},
		{name: "empty list", attachments: nil, wantParts: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parts := transform.ProcessAttachments(tt.attachments)
			assert.Len(t, parts, tt.wantParts)
			if tt.wantParts > 0 {
				mp, ok := parts[0].(agent.MediaPart)
				require.True(t, ok)
				assert.Equal(t, tt.wantKind, mp.Kind)
			}
		})
	}
}

func TestMediaPartConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parts   []agent.ContentPart
		wantErr string
		check   func(t *testing.T, parts []llms.ContentPart)
	}{
		{
			name:  "image url",
			parts: []agent.ContentPart{agent.MediaPart{Kind: agent.MediaKindImage, URL: "https://x/a.png", MIMEType: "image/png"}},
			check: func(t *testing.T, parts []llms.ContentPart) {
				t.Helper()
				require.Len(t, parts, 1)
				_, ok := parts[0].(llms.ImageURLContent)
				assert.True(t, ok)
			},
		},
		{
			name:  "image binary for anthropic path",
			parts: []agent.ContentPart{agent.MediaPart{Kind: agent.MediaKindImage, MIMEType: "image/png", Data: []byte{1, 2, 3}}},
			check: func(t *testing.T, parts []llms.ContentPart) {
				t.Helper()
				require.Len(t, parts, 1)
				_, ok := parts[0].(llms.BinaryContent)
				assert.True(t, ok)
			},
		},
		{
			name:    "audio without data rejected",
			parts:   []agent.ContentPart{agent.MediaPart{Kind: agent.MediaKindAudio, MIMEType: "audio/wav", URL: "https://x/a.wav"}},
			wantErr: "requires binary data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := transform.DefaultConvertToLLM([]agent.AgentMessage{
				agent.UserMessage{Parts: tt.parts},
			})
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, 1)
			tt.check(t, result[0].Parts)
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
