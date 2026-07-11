package chatagent

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// TokenUsageSourceFromRunKind maps a chat agent run kind to a persisted usage source.
func TokenUsageSourceFromRunKind(kind RunKind) string {
	switch kind {
	case RunKindPipeline:
		return types.TokenUsageSourcePipeline
	case RunKindScheduled:
		return types.TokenUsageSourceScheduledTask
	default:
		return types.TokenUsageSourceAgent
	}
}

// RecordLLMUsageMessages persists token usage from assistant messages.
func RecordLLMUsageMessages(ctx context.Context, uid types.Uid, sessionID, source string, messages []agent.AgentMessage) {
	if uid.IsZero() {
		return
	}
	usageStore := store.NewLLMUsageStoreFromDatabase()
	if usageStore == nil {
		return
	}
	source = types.NormalizeTokenUsageSource(source)
	for _, raw := range messages {
		assistant, ok := raw.(msg.AssistantMessage)
		if !ok || assistant.Usage == nil {
			continue
		}
		err := usageStore.RecordLLMUsage(ctx, &types.LLMUsageRecordInput{
			UID:              uid.String(),
			SessionID:        sessionID,
			Model:            assistant.Model,
			PromptTokens:     assistant.Usage.PromptTokens,
			CompletionTokens: assistant.Usage.CompletionTokens,
			TotalTokens:      assistant.Usage.TotalTokens,
			CacheRead:        assistant.Usage.CacheRead,
			CacheWrite:       assistant.Usage.CacheWrite,
			Source:           source,
		})
		if err != nil {
			flog.Warn("[chat-agent] record llm usage session=%s source=%s: %v", sessionID, source, err)
		}
	}
}
