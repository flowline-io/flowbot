package llm

import (
	"context"
	"fmt"
	"slices"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
)

// NewModel creates a langchaingo model from flowbot model configuration.
func NewModel(ctx context.Context, modelName string) (llms.Model, string, error) {
	cfg := resolveModel(modelName)
	if cfg.Provider == "" {
		return nil, "", fmt.Errorf("agent llm: unknown model %q", modelName)
	}

	switch cfg.Provider {
	case llm.ProviderOpenAI, llm.ProviderOpenAICompatible:
		opts := []openai.Option{openai.WithToken(cfg.ApiKey), openai.WithModel(modelName)}
		if cfg.BaseUrl != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseUrl))
		}
		model, err := openai.New(opts...)
		if err != nil {
			return nil, "", fmt.Errorf("agent llm: openai model: %w", err)
		}
		return model, modelName, nil
	case llm.ProviderAnthropic:
		model, err := anthropic.New(
			anthropic.WithToken(cfg.ApiKey),
			anthropic.WithModel(modelName),
		)
		if err != nil {
			return nil, "", fmt.Errorf("agent llm: anthropic model: %w", err)
		}
		return model, modelName, nil
	case llm.ProviderGemini:
		model, err := googleai.New(ctx, googleai.WithAPIKey(cfg.ApiKey), googleai.WithDefaultModel(modelName))
		if err != nil {
			return nil, "", fmt.Errorf("agent llm: gemini model: %w", err)
		}
		return model, modelName, nil
	default:
		return nil, "", fmt.Errorf("agent llm: unsupported provider %q", cfg.Provider)
	}
}

func resolveModel(modelName string) config.Model {
	for _, item := range config.App.Models {
		if slices.Contains(item.ModelNames, modelName) {
			return item
		}
	}
	return config.Model{}
}
