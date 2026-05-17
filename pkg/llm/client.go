package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
)

var models = make(map[string]config.Model)
var loadOnceModels = sync.Once{}

func GetModel(modelName string) config.Model {
	loadOnceModels.Do(func() {
		for i, item := range config.App.Models {
			for _, name := range item.ModelNames {
				models[name] = config.App.Models[i]
			}
		}
	})
	return models[modelName]
}

// Client wraps a Provider for model operations
type Client struct {
	provider Provider
	model    string
	timeout  time.Duration
}

// ChatModel creates a chat model client
func ChatModel(_ context.Context, modelName string) (*Client, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model or agent disabled")
	}

	m := GetModel(modelName)

	provider, err := NewProvider(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Client{
		provider: provider,
		model:    modelName,
		timeout:  10 * time.Minute,
	}, nil
}

// Generate generates an LLM response
func Generate(ctx context.Context, client *Client, messages []*Message) (*Message, error) {
	_, err := CountMessageTokens(messages)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	resp, err := client.provider.Generate(ctx, &GenerateRequest{
		Model:    client.model,
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}

	return resp, nil
}

// Stream streams an LLM response
func Stream(ctx context.Context, client *Client, messages []*Message) (<-chan *Message, error) {
	_, err := CountMessageTokens(messages)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	fragCh, err := client.provider.GenerateStream(ctx, &GenerateRequest{
		Model:    client.model,
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("llm stream failed: %w", err)
	}

	msgCh := make(chan *Message, 1)
	go func() {
		defer close(msgCh)
		var accumulated string
		for {
			select {
			case <-ctx.Done():
				return
			case frag, ok := <-fragCh:
				if !ok {
					return
				}
				if frag.Err != nil {
					return
				}
				accumulated += frag.Content
				if frag.Done {
					select {
					case msgCh <- &Message{Role: AssistantRole, Content: accumulated}:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()

	return msgCh, nil
}
