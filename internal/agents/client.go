package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/config"
)

var models = make(map[string]config.Model)
var loadOnceModels = sync.Once{}

// GetModel returns the model configuration.
func GetModel(modelName string) config.Model {
	loadOnceModels.Do(func() {
		for i, item := range config.App.Models {
			for _, name := range item.ModelNames {
				models[name] = config.App.Models[i]
			}
		}
	})
	return models[modelName]
}

// ChatModel creates a chat model instance.
func ChatModel(ctx context.Context, modelName string) (model.ToolCallingChatModel, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model or agent disabled")
	}
	timeout := 10 * time.Minute

	m := GetModel(modelName)
	switch m.Provider {
	case ProviderOpenAI:
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  m.ApiKey,
			Model:   modelName,
			Timeout: timeout,
		})
	case ProviderOpenAICompatible:
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			BaseURL: m.BaseUrl,
			APIKey:  m.ApiKey,
			Model:   modelName,
			Timeout: timeout,
		})
	}

	return nil, fmt.Errorf("model provider not found")
}

// Generate generates an LLM response.
func Generate(ctx context.Context, llm model.ToolCallingChatModel, in []*schema.Message) (*schema.Message, error) {
	_, err := CountMessageTokens(in)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	result, err := llm.Generate(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}
	return result, nil
}

// Stream streams an LLM response.
func Stream(ctx context.Context, llm model.ToolCallingChatModel, in []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	_, err := CountMessageTokens(in)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	result, err := llm.Stream(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}
	return result, nil
}
