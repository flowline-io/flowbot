package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

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
