package config

import (
	"fmt"
	"log"
	"slices"
	"strings"
)

// ChatAgentChatModel returns the configured chat model for the chat agent.
func ChatAgentChatModel() string {
	return App.ChatAgent.ChatModel
}

// ChatAgentEnabled reports whether the chat agent is on.
// Explicit enabled:false wins. Otherwise a non-empty chat_model is required.
// When enabled is unset, non-empty chat_model keeps the legacy on semantics.
func ChatAgentEnabled() bool {
	cfg := App.ChatAgent
	if cfg.Enabled != nil && !*cfg.Enabled {
		return false
	}
	return strings.TrimSpace(cfg.ChatModel) != ""
}

// ChatAgentUsesLegacyEnable reports whether enablement comes only from a non-empty chat_model.
func ChatAgentUsesLegacyEnable() bool {
	cfg := App.ChatAgent
	return cfg.Enabled == nil && strings.TrimSpace(cfg.ChatModel) != ""
}

// LogChatAgentEnabledCompatWarn prints a one-shot startup warning for legacy chat_model enablement.
func LogChatAgentEnabledCompatWarn() {
	if !ChatAgentUsesLegacyEnable() {
		return
	}
	log.Print("[config] deprecated: chat_agent.enabled is unset; non-empty chat_model enables the agent. Set chat_agent.enabled: true|false explicitly.")
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
	return ResolveChatAgentModelPair(ChatAgentChatModel(), App.ChatAgent.ToolModel)
}

// ResolveChatAgentModelPair validates chat and tool model names and reports dual routing.
func ResolveChatAgentModelPair(chat, tool string) (chatOut, toolOut string, dual bool, err error) {
	chat = strings.TrimSpace(chat)
	tool = strings.TrimSpace(tool)
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
