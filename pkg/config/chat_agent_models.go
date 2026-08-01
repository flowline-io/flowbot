package config

import (
	"fmt"
	"slices"
	"strings"
)

// ChatAgentChatModel returns the configured chat model for the chat agent.
func ChatAgentChatModel() string {
	return App.ChatAgent.ChatModel
}

// ChatAgentEnabled reports whether the chat agent is enabled with a configured chat model.
func ChatAgentEnabled() bool {
	return App.ChatAgent.ChatModel != ""
}

// ModelRegistered reports whether modelName appears in the configured models list.
func ModelRegistered(modelName string) bool {
	return ModelProviderFor(modelName) != ""
}

// ModelProviderFor returns the provider for a registered model name, or "" if unknown.
func ModelProviderFor(modelName string) string {
	return providerForModelInList(App.Models, modelName)
}

func providerForModelInList(models []Model, modelName string) string {
	for _, item := range models {
		if slices.Contains(item.ModelNames, modelName) {
			return item.Provider
		}
	}
	return ""
}

// ResolveChatAgentModels resolves chat and tool model names and whether dual routing applies.
// Dual mode is enabled when tool_model is non-empty.
func ResolveChatAgentModels() (chat, tool string, dual bool, err error) {
	chat = ChatAgentChatModel()
	tool = App.ChatAgent.ToolModel
	dual = tool != ""
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

// ResolveApprovalModel returns the aux reviewer model: approval_model → tool_model → chat_model.
func ResolveApprovalModel() string {
	if m := strings.TrimSpace(App.ChatAgent.ApprovalModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(App.ChatAgent.ToolModel); m != "" {
		return m
	}
	return ChatAgentChatModel()
}

// ChatAgentApprovalModeDefault returns the YAML default approval mode, or "manual".
func ChatAgentApprovalModeDefault() string {
	raw := strings.TrimSpace(App.ChatAgent.ApprovalMode)
	if raw == "" {
		return "manual"
	}
	return raw
}
