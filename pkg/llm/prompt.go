package llm

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
)

type ChatTemplate interface {
	Format(ctx context.Context, data map[string]any) ([]*Message, error)
}

type adkChatTemplate struct {
	systemMessage string
	withHistory   bool
}

func (t *adkChatTemplate) Format(ctx context.Context, data map[string]any) ([]*Message, error) {
	content := ""
	if c, ok := data["content"].(string); ok {
		content = c
	}

	var messages []*Message

	messages = append(messages, &Message{
		Role:    SystemRole,
		Content: t.systemMessage,
	})

	if t.withHistory {
		if chatHistory, ok := data["chat_history"].([]*Message); ok {
			messages = append(messages, chatHistory...)
		}
	}

	if content != "" {
		messages = append(messages, &Message{
			Role:    UserRole,
			Content: content,
		})
	}

	return messages, nil
}

func DefaultTemplate() ChatTemplate {
	return &adkChatTemplate{
		systemMessage: fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Flowbot.Language),
		withHistory:   true,
	}
}

func DefaultMultiChatTemplate() ChatTemplate {
	return &adkChatTemplate{
		systemMessage: fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Flowbot.Language),
		withHistory:   true,
	}
}

func BaseTemplate() ChatTemplate {
	return &adkChatTemplate{
		systemMessage: "You are a helpful assistant.",
		withHistory:   false,
	}
}
