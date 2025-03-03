package agents

import (
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/config"
)

func DefaultTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Agent.Language)),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("{content}"),
	)
}

func DefaultMultiChatTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(fmt.Sprintf("You are a helpful assistant. Please answer in %s.", config.App.Agent.Language)),
		schema.MessagesPlaceholder("chat_history", true),
	)
}
