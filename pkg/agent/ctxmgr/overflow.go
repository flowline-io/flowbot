package ctxmgr

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// ErrSummarizationFailed indicates compaction summarization could not complete.
var ErrSummarizationFailed = result.NewCompactionError("summarization_failed", "summarization failed", nil)

// ErrCompactionRequired indicates context still exceeds budget but no history could be compacted.
var ErrCompactionRequired = result.NewCompactionError("nothing_to_compact", "compaction required but no removable history", nil)

// ErrBranchSummaryAborted indicates branch summarization was cancelled.
var ErrBranchSummaryAborted = result.NewBranchSummaryError("aborted", "branch summarization aborted", nil)

// IsContextOverflowErr reports whether an LLM error indicates context window overflow.
func IsContextOverflowErr(err error) bool {
	return result.IsContextOverflowErr(err)
}

// WrapOverflowError wraps an underlying error as a typed OverflowError when overflow is detected.
func WrapOverflowError(err error) error {
	return result.WrapOverflowError(err)
}

// IsContextOverflowMessage reports whether an assistant message indicates overflow.
func IsContextOverflowMessage(message msg.AssistantMessage, contextWindow int) bool {
	if message.StopReason == "error" {
		text := message.TextContent()
		if result.IsNonOverflowText(text) {
			return false
		}
		if result.MatchesOverflowText(text) {
			return true
		}
	}
	if contextWindow <= 0 || message.Usage == nil {
		return false
	}
	inputTokens := message.Usage.PromptTokens + message.Usage.CacheRead
	if message.StopReason == "complete" && inputTokens > contextWindow {
		return true
	}
	if message.StopReason == "length" && message.Usage.CompletionTokens == 0 {
		if inputTokens >= int(float64(contextWindow)*0.99) {
			return true
		}
	}
	return false
}

// IsOverflowResult checks stream errors and the last assistant message for overflow.
func IsOverflowResult(err error, messages []msg.AgentMessage, contextWindow int) bool {
	if IsContextOverflowErr(err) {
		return true
	}
	for i := len(messages) - 1; i >= 0; i-- {
		assistant, ok := messages[i].(msg.AssistantMessage)
		if !ok {
			continue
		}
		return IsContextOverflowMessage(assistant, contextWindow)
	}
	return false
}

func normalizeSummary(text string) string {
	return strings.TrimSpace(text)
}
