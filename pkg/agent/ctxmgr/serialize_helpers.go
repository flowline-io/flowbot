package ctxmgr

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

func serializeUserMessage(parts []msg.ContentPart) string {
	text := textFromParts(parts)
	var mediaStubs []string
	for _, part := range parts {
		if mp, ok := part.(msg.MediaPart); ok {
			mediaStubs = append(mediaStubs, msg.MediaPlaceholder(mp.Kind))
		}
	}
	body := text
	if len(mediaStubs) > 0 {
		stub := strings.Join(mediaStubs, " ")
		if body == "" {
			body = stub
		} else {
			body = body + " " + stub
		}
	}
	if body == "" {
		return ""
	}
	return "[User]: " + body
}

func serializeAssistantMessage(m msg.AssistantMessage) string {
	var parts []string
	if text := m.TextContent(); text != "" {
		parts = append(parts, "[Assistant]: "+text)
	}
	if calls := formatToolCalls(m.ToolCalls()); calls != "" {
		parts = append(parts, "[Assistant tool calls]: "+calls)
	}
	return strings.Join(parts, "\n\n")
}

func serializeToolResultMessage(parts []msg.ContentPart) string {
	if text := textFromParts(parts); text != "" {
		return "[Tool result]: " + truncateForSummary(text, toolResultMaxChars)
	}
	return ""
}

func serializeOneMessage(message msg.AgentMessage) string {
	switch m := message.(type) {
	case msg.UserMessage:
		return serializeUserMessage(m.Parts)
	case msg.AssistantMessage:
		return serializeAssistantMessage(m)
	case msg.ToolResultMessage:
		return serializeToolResultMessage(m.Parts)
	case msg.CustomMessage:
		if m.ExcludeFromContext || m.DisplayOnly {
			return ""
		}
		return serializeUserMessage(m.Parts)
	case msg.BranchSummaryMessage:
		if m.Summary != "" {
			return "[User]: " + m.Summary
		}
	case msg.CompactionSummaryMessage:
		if m.Summary != "" {
			return "[User]: " + m.Summary
		}
	}
	return ""
}

func formatToolCalls(calls []msg.ToolCallPart) string {
	if len(calls) == 0 {
		return ""
	}
	formatted := make([]string, 0, len(calls))
	for _, call := range calls {
		formatted = append(formatted, fmt.Sprintf("%s(%s)", call.Name, call.Arguments))
	}
	return strings.Join(formatted, "; ")
}

func truncateForSummary(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	truncated := len(text) - maxChars
	return text[:maxChars] + fmt.Sprintf("\n\n[... %d more characters truncated]", truncated)
}
