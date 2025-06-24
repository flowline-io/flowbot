package reader

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/internal/agents"
)

func getAIResult(ctx context.Context, modelName, prompt, request string) (string, error) {
	messages, err := agents.DefaultTemplate().Format(ctx, map[string]any{
		"content": fmt.Sprintf("%s\n---\n%s", request, prompt),
	})
	if err != nil {
		return "", fmt.Errorf("%s bot, prompt format failed, %w", Name, err)
	}

	llm, err := agents.ChatModel(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("%s bot, chat model failed, %w", Name, err)
	}

	resp, err := agents.Generate(ctx, llm, messages)
	if err != nil {
		return "", fmt.Errorf("%s bot, llm generate failed, %w", Name, err)
	}

	return resp.Content, nil
}
