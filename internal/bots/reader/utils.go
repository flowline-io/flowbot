package reader

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func getAIResult(ctx context.Context, modelName, prompt, request string) (string, error) {
	messages, err := llm.DefaultTemplate().Format(ctx, map[string]any{
		"content": fmt.Sprintf("%s\n---\n%s", request, prompt),
	})
	if err != nil {
		return "", fmt.Errorf("%s bot, prompt format failed, %w", Name, err)
	}

	llmClient, err := llm.ChatModel(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("%s bot, chat model failed, %w", Name, err)
	}

	resp, err := llm.Generate(ctx, llmClient, messages)
	if err != nil {
		return "", fmt.Errorf("%s bot, llm generate failed, %w", Name, err)
	}

	return resp.Content, nil
}
