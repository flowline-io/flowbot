package agents

import (
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/config"
)

// DefaultTemplate returns the default single-turn conversation template.
func DefaultTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Flowbot.Language)),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("{content}"),
	)
}

// DefaultMultiChatTemplate returns the default multi-turn conversation template.
func DefaultMultiChatTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Flowbot.Language)),
		schema.MessagesPlaceholder("chat_history", true),
	)
}

// BaseTemplate returns the base template (without chat history).
func BaseTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("{content}"),
	)
}
