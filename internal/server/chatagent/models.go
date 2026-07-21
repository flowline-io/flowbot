package chatagent

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

// agentLoopConfig resolves chat agent models and builds loop configuration
// using global yaml defaults (no session override).
func agentLoopConfig() (cfg agent.Config, chatModel, toolModel string, dual bool, err error) {
	return agentLoopConfigForSession(context.Background(), "")
}

// agentLoopConfigForSession resolves chat agent models for sessionID.
// When the session has a non-empty model override that is registered, it
// replaces the global chat_model; the tool_model is always taken from yaml.
func agentLoopConfigForSession(ctx context.Context, sessionID string) (cfg agent.Config, chatModel, toolModel string, dual bool, err error) {
	chatModel, toolModel, dual, err = config.ResolveChatAgentModels()
	if err != nil {
		return agent.Config{}, "", "", false, fmt.Errorf("resolve chat agent models: %w", err)
	}
	cfg = agent.DefaultConfig()
	cfg.ModelName = chatModel
	if sessionID != "" {
		effective := ResolveEffectiveSessionSettings(ctx, sessionID)
		if effective.Model != "" {
			chatModel = effective.Model
			cfg.ModelName = chatModel
		}
		cfg.ThinkingLevel = effective.ThinkingLevel
	}
	cfg.MaxSteps = runMaxSteps()
	retry := agentllm.RetryConfigFromChatAgent(config.App.ChatAgent.LLMRetry)
	cfg.LLMRetryMaxAttempts = retry.MaxAttempts
	cfg.LLMRetryInitialInterval = retry.InitialInterval
	cfg.LLMRetryMaxInterval = retry.MaxInterval
	cfg.LLMRetryMultiplier = retry.Multiplier
	if dual {
		cfg.ChatModel = chatModel
		cfg.ToolModel = toolModel
	}
	return cfg, chatModel, toolModel, dual, nil
}

func runMaxSteps() int {
	maxSteps := config.App.ChatAgent.MaxSteps
	if maxSteps <= 0 {
		return 30
	}
	return maxSteps
}
