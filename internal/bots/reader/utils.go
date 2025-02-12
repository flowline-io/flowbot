package reader

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers"
	openaiProvider "github.com/flowline-io/flowbot/pkg/providers/openai"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	rssClient "miniflux.app/v2/client"
)

func entriyFilter(entry *rssClient.Entry) bool {
	// todo allow_list
	// todo deny_list
	return false
}

func getAIResult(prompt, request string) (string, error) {
	tokenVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.TokenKey)
	baseUrlVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.BaseUrlKey)
	modelVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.ModelKey)

	llm, err := openai.New(
		openai.WithToken(tokenVal.String()),
		openai.WithBaseURL(baseUrlVal.String()),
		openai.WithModel(modelVal.String()),
	)
	if err != nil {
		return "", fmt.Errorf("openai new failed, %w", err)
	}

	messages := []llms.MessageContent{
		// {"role": "system", "content": "You are a helpful assistant."},
		// {"role": "user", "content": request + "\n---\n" + prompt},
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextContent{Text: "You are a helpful assistant."}},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: request + "\n---\n" + prompt}},
		},
	}

	resp, err := llm.GenerateContent(context.Background(), messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate content, %w", err)
	}

	choices := resp.Choices
	if len(choices) < 1 {
		return "", errors.New("empty response from model")
	}
	c1 := choices[0]
	return c1.Content, nil
}
