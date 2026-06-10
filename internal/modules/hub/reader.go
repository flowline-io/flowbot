package hub

import (
	"context"
	"fmt"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
)

func getAIResult(ctx context.Context, modelName, prompt, request string) (string, error) {
	content, err := agentllm.GenerateWithTemplate(ctx, modelName, agentllm.DefaultTemplate(), map[string]any{
		"content": fmt.Sprintf("%s\n---\n%s", request, prompt),
	})
	if err != nil {
		return "", fmt.Errorf("%s module, llm generate failed, %w", Name, err)
	}
	return content, nil
}
