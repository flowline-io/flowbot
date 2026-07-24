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
			content, err := userToLLM(m)
			if err != nil {
				return nil, err
			}
			result = append(result, content)
		case msg.AssistantMessage:
			content, err := assistantToLLM(m)
			if err != nil {
				return nil, err
			}
			result = append(result, content)
		case msg.ToolResultMessage:
			result = append(result, toolResultToLLM(m))
		case msg.CustomMessage:
			if m.ExcludeFromContext || m.DisplayOnly {
				continue
			}
			content, err := customToLLM(m)
			if err != nil {
				return nil, err
			}
			result = append(result, content)
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

func userToLLM(message msg.UserMessage) (llms.MessageContent, error) {
	parts, err := partsToLLM(message.Parts)
	if err != nil {
		return llms.MessageContent{}, err
	}
	return llms.MessageContent{Role: llms.ChatMessageTypeHuman, Parts: parts}, nil
}

func assistantToLLM(message msg.AssistantMessage) (llms.MessageContent, error) {
	parts, err := partsToLLM(message.Parts)
	if err != nil {
		return llms.MessageContent{}, err
	}
	return llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: parts}, nil
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

func customToLLM(message msg.CustomMessage) (llms.MessageContent, error) {
	parts, err := partsToLLM(message.Parts)
	if err != nil {
		return llms.MessageContent{}, err
	}
	return llms.MessageContent{Role: llms.ChatMessageTypeHuman, Parts: parts}, nil
}

func partsToLLM(parts []msg.ContentPart) ([]llms.ContentPart, error) {
	result := make([]llms.ContentPart, 0, len(parts))
	for _, part := range parts {
		switch p := part.(type) {
		case msg.TextPart:
			result = append(result, llms.TextPart(p.Text))
		case msg.MediaPart:
			llmPart, err := mediaPartToLLM(p)
			if err != nil {
				return nil, err
			}
			result = append(result, llmPart)
		case msg.ToolCallPart:
			result = append(result, llms.ToolCall{
				ID:   msg.EnsureToolCallID(p.ID),
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      p.Name,
					Arguments: p.Arguments,
				},
			})
		}
	}
	return result, nil
}

func mediaPartToLLM(p msg.MediaPart) (llms.ContentPart, error) {
	if p.Kind != "" && p.Kind != msg.MediaKindImage {
		if len(p.Data) == 0 {
			return nil, fmt.Errorf("transform: non-image media kind %q requires binary data", p.Kind)
		}
		return llms.BinaryPart(p.MIMEType, p.Data), nil
	}
	if len(p.Data) > 0 {
		return llms.BinaryPart(p.MIMEType, p.Data), nil
	}
	if p.URL != "" {
		return llms.ImageURLPart(p.URL), nil
	}
	return nil, fmt.Errorf("transform: image media part requires URL or data (file_id=%q)", p.FileID)
}

func textFromParts(parts []msg.ContentPart) string {
	var text strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = text.WriteString(tp.Text)
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
