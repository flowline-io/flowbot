package transform

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/tmc/langchaingo/llms"
)

const (
	compactionSummaryPrefix = "The conversation history before this point was compacted into the following summary:\n\n<summary>\n"
	compactionSummarySuffix = "\n</summary>"
	branchSummaryPrefix     = "The following is a summary of a branch that this conversation came back from:\n\n<summary>\n"
	branchSummarySuffix     = "</summary>"
)

// DefaultConvertToLLM converts standard agent messages into langchaingo messages.
func DefaultConvertToLLM(messages []msg.AgentMessage) ([]llms.MessageContent, error) {
	result := make([]llms.MessageContent, 0, len(messages))
	for _, message := range messages {
		switch m := message.(type) {
		case msg.UserMessage:
			result = append(result, userToLLM(m))
		case msg.AssistantMessage:
			result = append(result, assistantToLLM(m))
		case msg.ToolResultMessage:
			result = append(result, toolResultToLLM(m))
		case msg.CustomMessage:
			if m.ExcludeFromContext || m.DisplayOnly {
				continue
			}
			result = append(result, customToLLM(m))
		case msg.BranchSummaryMessage:
			result = append(result, llms.TextParts(llms.ChatMessageTypeHuman, branchSummaryPrefix+m.Summary+branchSummarySuffix))
		case msg.CompactionSummaryMessage:
			result = append(result, llms.TextParts(llms.ChatMessageTypeHuman, compactionSummaryPrefix+m.Summary+compactionSummarySuffix))
		default:
			return nil, fmt.Errorf("transform: unsupported message type %T", message)
		}
	}
	return result, nil
}

func userToLLM(message msg.UserMessage) llms.MessageContent {
	parts := partsToLLM(message.Parts)
	return llms.MessageContent{Role: llms.ChatMessageTypeHuman, Parts: parts}
}

func assistantToLLM(message msg.AssistantMessage) llms.MessageContent {
	parts := partsToLLM(message.Parts)
	return llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: parts}
}

func toolResultToLLM(message msg.ToolResultMessage) llms.MessageContent {
	content := textFromParts(message.Parts)
	return llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: message.ToolCallID,
				Name:       message.Name,
				Content:    content,
			},
		},
	}
}

func customToLLM(message msg.CustomMessage) llms.MessageContent {
	return llms.MessageContent{Role: llms.ChatMessageTypeHuman, Parts: partsToLLM(message.Parts)}
}

func partsToLLM(parts []msg.ContentPart) []llms.ContentPart {
	result := make([]llms.ContentPart, 0, len(parts))
	for _, part := range parts {
		switch p := part.(type) {
		case msg.TextPart:
			result = append(result, llms.TextPart(p.Text))
		case msg.ImagePart:
			if p.URL != "" {
				result = append(result, llms.ImageURLPart(p.URL))
			} else {
				result = append(result, llms.BinaryPart(p.MIMEType, p.Data))
			}
		case msg.ToolCallPart:
			result = append(result, llms.ToolCall{
				ID:   p.ID,
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      p.Name,
					Arguments: p.Arguments,
				},
			})
		}
	}
	return result
}

func textFromParts(parts []msg.ContentPart) string {
	var text strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			text.WriteString(tp.Text)
		}
	}
	return text.String()
}

// FilterContext returns messages unchanged; callers may replace this hook.
func FilterContext(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
	return messages, nil
}

// MergeSystemPrompt prepends a system prompt into agent context without duplicating messages.
func MergeSystemPrompt(base, extra string) string {
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + "\n\n" + extra
}
