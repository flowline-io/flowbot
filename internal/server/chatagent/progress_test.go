package chatagent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatParseProgressMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   chatagent.ProgressArtifact
		want chatagent.ProgressArtifact
	}{
		{
			name: "round trip with items",
			in: chatagent.ProgressArtifact{
				Goal: "ship harness",
				Done: []string{"retry", "metrics"},
				Next: []string{"eval"},
			},
			want: chatagent.ProgressArtifact{
				Goal: "ship harness",
				Done: []string{"retry", "metrics"},
				Next: []string{"eval"},
			},
		},
		{
			name: "empty sections",
			in:   chatagent.ProgressArtifact{},
			want: chatagent.ProgressArtifact{},
		},
		{
			name: "goal only",
			in:   chatagent.ProgressArtifact{Goal: "fix bug"},
			want: chatagent.ProgressArtifact{Goal: "fix bug"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.ParseProgressMarkdown(chatagent.FormatProgressMarkdown(tt.in))
			assert.Equal(t, tt.want.Goal, got.Goal)
			assert.Equal(t, tt.want.Done, got.Done)
			assert.Equal(t, tt.want.Next, got.Next)
		})
	}
}

func TestTruncateProgressSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		maxTokens int
		wantEmpty bool
		wantMax   int
	}{
		{name: "under cap unchanged", text: "short", maxTokens: 100, wantMax: 100},
		{name: "zero cap empty", text: "hello", maxTokens: 0, wantEmpty: true},
		{name: "over cap truncated", text: strings.Repeat("abcd ", 400), maxTokens: 20, wantMax: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.TruncateProgressSummary(tt.text, tt.maxTokens)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			assert.LessOrEqual(t, chatagent.EstimateTextTokens(got), tt.wantMax)
			if chatagent.EstimateTextTokens(tt.text) <= tt.maxTokens {
				assert.Equal(t, tt.text, got)
			} else {
				assert.Less(t, len(got), len(tt.text))
			}
		})
	}
}

func TestDeriveProgressFromMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []msg.AgentMessage
		wantGoal string
		wantDone int
		wantNext bool
	}{
		{
			name: "user goal and tool done",
			messages: []msg.AgentMessage{
				msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "build eval suite"}}},
				msg.ToolResultMessage{Name: "write_file", IsError: false},
			},
			wantGoal: "build eval suite",
			wantDone: 1,
			wantNext: true,
		},
		{
			name: "assistant without tools awaits input",
			messages: []msg.AgentMessage{
				msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
				msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "done"}}},
			},
			wantGoal: "hello",
			wantNext: true,
		},
		{
			name:     "empty messages",
			messages: nil,
			wantGoal: "",
			wantDone: 0,
			wantNext: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.DeriveProgressFromMessages(tt.messages)
			assert.Equal(t, tt.wantGoal, got.Goal)
			assert.Len(t, got.Done, tt.wantDone)
			if tt.wantNext {
				assert.NotEmpty(t, got.Next)
			} else {
				assert.Empty(t, got.Next)
			}
		})
	}
}

func TestRegisterHooksProgressInject(t *testing.T) {
	prev := config.App.ChatAgent.Workspace
	t.Cleanup(func() { config.App.ChatAgent.Workspace = prev })

	tests := []struct {
		name         string
		messages     []msg.AgentMessage
		wantInjected bool
		wantFile     bool
	}{
		{
			name: "injects progress custom message",
			messages: []msg.AgentMessage{
				msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "implement progress"}}},
			},
			wantInjected: true,
			wantFile:     true,
		},
		{
			name: "strips previous progress before reinject",
			messages: []msg.AgentMessage{
				msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "goal"}}},
				msg.CustomMessage{CustomType: "progress", Parts: []msg.ContentPart{msg.TextPart{Text: "old"}}},
			},
			wantInjected: true,
			wantFile:     true,
		},
		{
			name:         "empty messages still writes default artifact",
			messages:     nil,
			wantInjected: true,
			wantFile:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			config.App.ChatAgent.Workspace = root
			reg := hooks.NewRegistry()
			chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{SessionID: "progress-test"})
			out, err := reg.EmitContext(context.Background(), tt.messages)
			require.NoError(t, err)
			if tt.wantInjected {
				found := false
				for _, item := range out {
					custom, ok := item.(msg.CustomMessage)
					if ok && custom.CustomType == "progress" {
						found = true
						assert.LessOrEqual(t, chatagent.EstimateTextTokens(chatagentText(custom)), chatagent.ProgressTokenCap)
					}
				}
				assert.True(t, found)
			}
			if tt.wantFile {
				_, err := os.Stat(filepath.Join(root, chatagent.ProgressRelPath))
				assert.NoError(t, err)
			}
		})
	}
}

func chatagentText(custom msg.CustomMessage) string {
	var b strings.Builder
	for _, part := range custom.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = b.WriteString(tp.Text)
		}
	}
	return b.String()
}
