package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTokenUsageSourceFromRunKind(t *testing.T) {
	tests := []struct {
		name string
		kind RunKind
		want string
	}{
		{name: "interactive defaults to agent", kind: RunKindInteractive, want: types.TokenUsageSourceAgent},
		{name: "pipeline source", kind: RunKindPipeline, want: types.TokenUsageSourcePipeline},
		{name: "scheduled source", kind: RunKindScheduled, want: types.TokenUsageSourceScheduledTask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TokenUsageSourceFromRunKind(tt.kind))
		})
	}
}

func TestRecordLLMUsageMessages(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	uid := types.Uid("user-usage")
	sessionID := "sess-usage"

	tests := []struct {
		name     string
		uid      types.Uid
		messages []agent.AgentMessage
	}{
		{
			name: "records assistant usage",
			uid:  uid,
			messages: []agent.AgentMessage{
				msg.AssistantMessage{
					Model: "gpt-test",
					Usage: &msg.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
				},
			},
		},
		{
			name: "skips non-assistant messages",
			uid:  uid,
			messages: []agent.AgentMessage{
				msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}},
			},
		},
		{
			name:     "zero uid is no-op",
			uid:      types.Uid(""),
			messages: []agent.AgentMessage{msg.AssistantMessage{Usage: &msg.Usage{TotalTokens: 1}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				RecordLLMUsageMessages(ctx, tt.uid, sessionID, types.TokenUsageSourceAgent, tt.messages)
			})
		})
	}
}
