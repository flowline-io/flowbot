package llm

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"
)

// AgentModelName returns the configured model for a named agent when enabled.
func AgentModelName(name string) string {
	if name == "chat" {
		if !config.ChatAgentEnabled() {
			return ""
		}
		return config.ChatAgentChatModel()
	}
	for _, item := range config.App.Agents {
		if item.Name == name && item.Enabled {
			return item.Model
		}
	}
	return ""
}

// AgentEnabled reports whether a named agent is configured with a model and enabled.
func AgentEnabled(name string) bool {
	if name == "chat" {
		return config.ChatAgentEnabled()
	}
	for _, item := range config.App.Agents {
		if item.Name == name && item.Enabled && item.Model != "" {
			return true
		}
	}
	return false
}

// LLMGenerate performs a single-shot completion using BaseTemplate and the given model.
func LLMGenerate(ctx context.Context, modelName, prompt string) (string, error) {
	return GenerateWithTemplate(ctx, modelName, BaseTemplate(), map[string]any{
		"content": prompt,
	})
}

// GenerateWithTemplate performs a single-shot completion using the given prompt template.
func GenerateWithTemplate(ctx context.Context, modelName string, template ChatTemplate, data map[string]any) (string, error) {
	if modelName == "" {
		return "", fmt.Errorf("agent llm: model or agent disabled")
	}

	messages, err := template.Format(ctx, data)
	if err != nil {
		return "", fmt.Errorf("agent llm: prompt format: %w", err)
	}

	model, resolvedName, err := NewModel(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("agent llm: chat model: %w", err)
	}

	return generateWithModel(ctx, model, resolvedName, messages)
}

func generateWithModel(
	ctx context.Context,
	model llms.Model,
	modelName string,
	messages []llms.MessageContent,
) (string, error) {
	content, err := Complete(ctx, model, "", messages, modelName, 0)
	if err != nil {
		return "", fmt.Errorf("agent llm: generate: %w", err)
	}

	return content, nil
}
