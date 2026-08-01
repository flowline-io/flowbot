package chatagent

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"
)

const defaultApprovalTimeout = 10 * time.Second

// llmCompleter adapts langchaingo models to approval.Completer.
type llmCompleter struct {
	model     llms.Model
	modelName string
}

func (c llmCompleter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, userPrompt)}
	return agentllm.Complete(ctx, c.model, systemPrompt, messages, c.modelName, 256)
}

// NewApprovalReviewer builds an aux security reviewer for auto mode.
// Returns nil when no approval/chat model is configured.
func NewApprovalReviewer(ctx context.Context) (approval.Reviewer, error) {
	modelName := config.ResolveApprovalModel()
	if modelName == "" {
		return nil, fmt.Errorf("approval model unavailable")
	}
	model, resolved, err := NewModelForTest(ctx, modelName)
	if err != nil {
		return nil, err
	}
	return &approval.LLMReviewer{Complete: llmCompleter{model: model, modelName: resolved}}, nil
}

func approvalReviewTimeout() time.Duration {
	if d := config.App.ChatAgent.ApprovalTimeout; d > 0 {
		return d
	}
	return defaultApprovalTimeout
}

func approvalDenialThreshold() int {
	if n := config.App.ChatAgent.ApprovalDenialThreshold; n > 0 {
		return n
	}
	return approval.DefaultDenialThreshold
}
