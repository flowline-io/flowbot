package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/flowline-io/flowbot/pkg/config"
)

const (
	ProviderOpenAI           = "openai"
	ProviderOpenAICompatible = "openai-compatible"
	ProviderOllama           = "ollama"
)

var models = make(map[string]config.Model)
var loadOnceModels = sync.Once{}

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
	case ProviderOllama:
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: m.BaseUrl,
			Model:   modelName,
			Timeout: timeout,
		})
	}

	return nil, fmt.Errorf("model provider not found")
}
