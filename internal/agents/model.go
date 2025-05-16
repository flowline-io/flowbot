package agents

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"sync"
	"time"
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
	m := GetModel(modelName)
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: m.BaseUrl,
		APIKey:  m.ApiKey,
		Model:   modelName,
		Timeout: 10 * time.Minute,
	})
}
