package ctxmgr

import (
	"math"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const estimatedImageChars = 4800

// ContextUsageEstimate is a token budget estimate for a message list.
type ContextUsageEstimate struct {
	Tokens         int
	UsageTokens    int
	TrailingTokens int
	LastUsageIndex int
}

// CalculateContextTokens returns total tokens from provider usage metadata.
func CalculateContextTokens(usage msg.Usage) int {
	if usage.TotalTokens > 0 {
		return usage.TotalTokens
	}
	return usage.PromptTokens + usage.CompletionTokens + usage.CacheRead + usage.CacheWrite
}

// EstimateTokens conservatively estimates token count for one agent message.
func EstimateTokens(message msg.AgentMessage) int {
	switch m := message.(type) {
	case msg.UserMessage:
		return ceilDiv(estimatePartsChars(m.Parts), 4)
	case msg.AssistantMessage:
		chars := estimatePartsChars(m.Parts)
		return ceilDiv(chars, 4)
	case msg.ToolResultMessage:
		return ceilDiv(estimatePartsChars(m.Parts), 4)
	case msg.CustomMessage:
		if m.ExcludeFromContext || m.DisplayOnly {
			return 0
		}
		return ceilDiv(estimatePartsChars(m.Parts), 4)
	case msg.BranchSummaryMessage:
		return ceilDiv(len(m.Summary), 4)
	case msg.CompactionSummaryMessage:
		return ceilDiv(len(m.Summary), 4)
	default:
		return 0
	}
}

// EstimateContextTokens estimates total context tokens, preferring the last assistant usage.
func EstimateContextTokens(messages []msg.AgentMessage) ContextUsageEstimate {
	usageInfo := lastAssistantUsageInfo(messages)
	if usageInfo == nil {
		estimated := 0
		for _, message := range messages {
			estimated += EstimateTokens(message)
		}
		return ContextUsageEstimate{
			Tokens:         estimated,
			TrailingTokens: estimated,
			LastUsageIndex: -1,
		}
	}

	usageTokens := CalculateContextTokens(*usageInfo.usage)
	trailing := 0
	for i := usageInfo.index + 1; i < len(messages); i++ {
		trailing += EstimateTokens(messages[i])
	}
	return ContextUsageEstimate{
		Tokens:         usageTokens + trailing,
		UsageTokens:    usageTokens,
		TrailingTokens: trailing,
		LastUsageIndex: usageInfo.index,
	}
}

type assistantUsageInfo struct {
	usage *msg.Usage
	index int
}

func lastAssistantUsageInfo(messages []msg.AgentMessage) *assistantUsageInfo {
	for i := len(messages) - 1; i >= 0; i-- {
		assistant, ok := messages[i].(msg.AssistantMessage)
		if !ok {
			continue
		}
		if assistant.StopReason == "aborted" || assistant.StopReason == "error" {
			continue
		}
		if assistant.Usage == nil {
			continue
		}
		return &assistantUsageInfo{usage: assistant.Usage, index: i}
	}
	return nil
}

func estimatePartsChars(parts []msg.ContentPart) int {
	chars := 0
	for _, part := range parts {
		switch p := part.(type) {
		case msg.TextPart:
			chars += len(p.Text)
		case msg.ImagePart:
			chars += estimatedImageChars
		case msg.ToolCallPart:
			chars += len(p.Name) + len(p.Arguments)
		}
	}
	return chars
}

func ceilDiv(n, d int) int {
	if d <= 0 {
		return n
	}
	return int(math.Ceil(float64(n) / float64(d)))
}

func textFromParts(parts []msg.ContentPart) string {
	var b strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = b.WriteString(tp.Text)
		}
	}
	return b.String()
}
