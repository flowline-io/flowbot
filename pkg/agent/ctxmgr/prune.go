package ctxmgr

import "github.com/flowline-io/flowbot/pkg/agent/msg"

const (
	pruneProtectTokens = 40000
	pruneMinimumTokens = 20000
)

// PruneToolOutputs removes older tool result payloads while keeping recent tool output intact.
func PruneToolOutputs(messages []msg.AgentMessage, settings Settings) []msg.AgentMessage {
	if !settings.PruneToolOutputs || len(messages) == 0 {
		return messages
	}

	kept := make([]msg.AgentMessage, 0, len(messages))
	toolTokensKept := 0
	toolTokensPruned := 0

	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		toolResult, ok := message.(msg.ToolResultMessage)
		if !ok {
			kept = append(kept, message)
			continue
		}

		tokens := EstimateTokens(toolResult)
		if toolTokensKept+tokens <= pruneProtectTokens {
			toolTokensKept += tokens
			kept = append(kept, message)
			continue
		}
		toolTokensPruned += tokens
	}

	if toolTokensPruned < pruneMinimumTokens {
		return messages
	}

	for left, right := 0, len(kept)-1; left < right; left, right = left+1, right-1 {
		kept[left], kept[right] = kept[right], kept[left]
	}
	return kept
}
