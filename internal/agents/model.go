package agents

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/flowline-io/flowbot/pkg/config"
)

func ChatModel(ctx context.Context, modelName string) (model.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: config.App.Agent.BaseUrl,
		APIKey:  config.App.Agent.Token,
		Model:   modelName,
		Timeout: time.Minute,
	})
}

func Model() string {
	return config.App.Agent.Model
}

func ToolcallModel() string {
	return config.App.Agent.ToolModel
}
