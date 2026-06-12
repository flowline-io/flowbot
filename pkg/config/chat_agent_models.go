package config

import (
	"fmt"
	"slices"
)

const chatAgentName = "chat"

// ChatAgentChatModel returns the resolved chat model for the chat agent.
// chat_agent.chat_model overrides agents.chat.model when set.
func ChatAgentChatModel() string {
	if chat := App.ChatAgent.ChatModel; chat != "" {
		return chat
	}
	return agentModelName(chatAgentName)
}

// ChatAgentEnabled reports whether the chat agent is enabled with a resolvable chat model.
func ChatAgentEnabled() bool {
	for _, item := range App.Agents {
		if item.Name != chatAgentName {
			continue
		}
		return item.Enabled && ChatAgentChatModel() != ""
	}
	return false
}

// ModelRegistered reports whether modelName appears in the configured models list.
func ModelRegistered(modelName string) bool {
	return ModelProviderFor(modelName) != ""
}

// ModelProviderFor returns the provider for a registered model name, or "" if unknown.
func ModelProviderFor(modelName string) string {
	for _, item := range App.Models {
		if slices.Contains(item.ModelNames, modelName) {
			return item.Provider
		}
	}
	return ""
}

// ResolveChatAgentModels resolves chat and tool model names and whether dual routing applies.
// Chat defaults to agents.chat.model when chat_agent.chat_model is empty.
// Dual mode requires both names to be non-empty and different, registered, and same provider.
func ResolveChatAgentModels() (chat, tool string, dual bool, err error) {
	chat = ChatAgentChatModel()
	tool = App.ChatAgent.ToolModel
	dual = chat != "" && tool != "" && chat != tool
	if !dual {
		if chat != "" && !ModelRegistered(chat) {
			return "", "", false, fmt.Errorf("chat model %q is not registered in models", chat)
		}
		return chat, tool, false, nil
	}
	if !ModelRegistered(chat) {
		return "", "", false, fmt.Errorf("chat model %q is not registered in models", chat)
	}
	if !ModelRegistered(tool) {
		return "", "", false, fmt.Errorf("tool model %q is not registered in models", tool)
	}
	chatProvider := ModelProviderFor(chat)
	toolProvider := ModelProviderFor(tool)
	if chatProvider != toolProvider {
		return "", "", false, fmt.Errorf(
			"chat model %q (provider %q) and tool model %q (provider %q) must use the same provider",
			chat, chatProvider, tool, toolProvider,
		)
	}
	return chat, tool, true, nil
}

func agentModelName(name string) string {
	for _, item := range App.Agents {
		if item.Name == name && item.Enabled {
			return item.Model
		}
	}
	return ""
}
