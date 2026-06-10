package llm

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"
)

// ChatTemplate formats prompt data into langchaingo messages.
type ChatTemplate interface {
	Format(ctx context.Context, data map[string]any) ([]llms.MessageContent, error)
}

type chatTemplate struct {
	systemMessage string
	withHistory   bool
}

func (t *chatTemplate) Format(_ context.Context, data map[string]any) ([]llms.MessageContent, error) {
	content := ""
	if c, ok := data["content"].(string); ok {
		content = c
	}

	messages := make([]llms.MessageContent, 0, 4)
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, t.systemMessage))

	if t.withHistory {
		if raw, ok := data["chat_history"]; ok {
			chatHistory, ok := raw.([]llms.MessageContent)
			if !ok {
				return nil, fmt.Errorf("agent llm: chat_history must be []llms.MessageContent")
			}
			messages = append(messages, chatHistory...)
		}
	}

	if content != "" {
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, content))
	}

	return messages, nil
}

// DefaultTemplate returns a chat template with language-aware system prompt and history support.
func DefaultTemplate() ChatTemplate {
	return &chatTemplate{
		systemMessage: fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Flowbot.Language),
		withHistory:   true,
	}
}

// DefaultMultiChatTemplate returns the same template as DefaultTemplate for multi-turn chat.
func DefaultMultiChatTemplate() ChatTemplate {
	return DefaultTemplate()
}

// BaseTemplate returns a minimal template without chat history for single-shot tasks.
func BaseTemplate() ChatTemplate {
	return &chatTemplate{
		systemMessage: "You are a helpful assistant.",
		withHistory:   false,
	}
}
