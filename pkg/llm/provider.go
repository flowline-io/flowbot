package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
)

var ErrUnknownProvider = errors.New("unknown llm provider")

// GenerateRequest holds parameters for a model generation call
type GenerateRequest struct {
	Model       string
	Messages    []*Message
	Tools       []Tool
	MaxTokens   int
	Temperature float64
}

// MessageFragment represents a partial streaming response chunk
type MessageFragment struct {
	Content string
	Done    bool
	Err     error
}

// Provider is the interface for LLM model backends
type Provider interface {
	Name() string
	Generate(ctx context.Context, req *GenerateRequest) (*Message, error)
	GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan MessageFragment, error)
}

// providerFactory maps provider names to constructors
type providerFactory func(cfg config.Model) (Provider, error)

var providers = make(map[string]providerFactory)

func register(name string, fn providerFactory) {
	providers[name] = fn
}

// NewProvider creates a provider based on model configuration
func NewProvider(cfg config.Model) (Provider, error) {
	fn, ok := providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, cfg.Provider)
	}
	return fn(cfg)
}
