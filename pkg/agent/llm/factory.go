package llm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
)

// defaultGeminiOpenAIBaseURL is Google's OpenAI-compatible Gemini endpoint.
// See https://ai.google.dev/gemini-api/docs/openai
const defaultGeminiOpenAIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai/"

type pooledModel struct {
	model    llms.Model
	name     string
	provider string
}

type modelCreatorFn func(ctx context.Context, modelName string) (llms.Model, string, error)

var (
	modelPool      sync.Map
	modelCreatorMu sync.RWMutex
	modelCreator   modelCreatorFn = NewModel
)

// SetModelCreatorForTest overrides model construction used by GetOrCreateModel.
func SetModelCreatorForTest(fn modelCreatorFn) {
	modelCreatorMu.Lock()
	defer modelCreatorMu.Unlock()
	if fn == nil {
		modelCreator = NewModel
		return
	}
	modelCreator = fn
}

// ResetModelPoolForTest clears cached langchaingo models.
func ResetModelPoolForTest() {
	modelPool = sync.Map{}
	modelCreatorMu.Lock()
	modelCreator = NewModel
	modelCreatorMu.Unlock()
}

// GetOrCreateModel returns a cached langchaingo model for the given model name.
func GetOrCreateModel(ctx context.Context, modelName string) (llms.Model, string, error) {
	if cached, ok := modelPool.Load(modelName); ok {
		if entry, ok := cached.(pooledModel); ok {
			return entry.model, entry.name, nil
		}
		modelPool.Delete(modelName)
	}
	modelCreatorMu.RLock()
	creator := modelCreator
	modelCreatorMu.RUnlock()
	model, resolvedName, err := creator(ctx, modelName)
	if err != nil {
		return nil, "", err
	}
	provider := resolveModel(modelName).Provider
	actual, loaded := modelPool.LoadOrStore(modelName, pooledModel{model: model, name: resolvedName, provider: provider})
	if loaded {
		if entry, ok := actual.(pooledModel); ok {
			return entry.model, entry.name, nil
		}
		modelPool.Delete(modelName)
	}
	return model, resolvedName, nil
}

// NewModel creates a langchaingo model from flowbot model configuration.
// Provider reasoning is enabled per request in StreamAssistant via ReasoningCallOptions
// because langchaingo exposes extended thinking through GenerateContent call options.
// Gemini uses Google's OpenAI-compatible HTTP endpoint (not llms/googleai) to avoid
// pulling cloud.google.com/go and vertexai into the runtime binary.
func NewModel(_ context.Context, modelName string) (llms.Model, string, error) {
	cfg := resolveModel(modelName)
	if cfg.Provider == "" {
		return nil, "", fmt.Errorf("agent llm: unknown model %q", modelName)
	}

	switch cfg.Provider {
	case ProviderOpenAI, ProviderOpenAICompatible, ProviderGemini:
		opts := []openai.Option{openai.WithToken(cfg.ApiKey), openai.WithModel(modelName)}
		if baseURL := openaiCompatibleBaseURL(cfg); baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}
		opts = append(opts, openai.WithHTTPClient(openaiHTTPClient(needsThinkingHTTPClient(modelName))))
		model, err := openai.New(opts...)
		if err != nil {
			return nil, "", fmt.Errorf("agent llm: openai model: %w", err)
		}
		return model, modelName, nil
	case ProviderAnthropic:
		opts := []anthropic.Option{
			anthropic.WithToken(cfg.ApiKey),
			anthropic.WithModel(modelName),
		}
		if cfg.BaseUrl != "" {
			opts = append(opts, anthropic.WithBaseURL(cfg.BaseUrl))
		}
		model, err := anthropic.New(opts...)
		if err != nil {
			return nil, "", fmt.Errorf("agent llm: anthropic model: %w", err)
		}
		return model, modelName, nil
	default:
		return nil, "", fmt.Errorf("agent llm: unsupported provider %q", cfg.Provider)
	}
}

// openaiCompatibleBaseURL returns the chat-completions base URL for OpenAI-family providers.
func openaiCompatibleBaseURL(cfg config.Model) string {
	if cfg.BaseUrl != "" {
		return cfg.BaseUrl
	}
	if cfg.Provider == ProviderGemini {
		return defaultGeminiOpenAIBaseURL
	}
	return ""
}

// OpenAICompatibleBaseURLForTest exposes openaiCompatibleBaseURL for unit tests.
func OpenAICompatibleBaseURLForTest(cfg config.Model) string {
	return openaiCompatibleBaseURL(cfg)
}

// ProviderForModel returns the configured provider name for a model.
func ProviderForModel(modelName string) string {
	if cached, ok := modelPool.Load(modelName); ok {
		if entry, ok := cached.(pooledModel); ok && entry.provider != "" {
			return entry.provider
		}
	}
	return resolveModel(modelName).Provider
}

func resolveModel(modelName string) config.Model {
	for _, item := range config.App.Models {
		if slices.Contains(item.ModelNames, modelName) {
			return item
		}
	}
	return config.Model{}
}
