package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"google.golang.org/genai"
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

// GenaiClient wraps genai.Client for model operations
type GenaiClient struct {
	client   *genai.Client
	model    string
	provider string
	baseURL  string
	apiKey   string
	timeout  time.Duration
}

// Generate generates content using the model
func (c *GenaiClient) Generate(ctx context.Context, contents []*genai.Content) (*genai.Content, error) {
	if c.client == nil {
		return nil, fmt.Errorf("genai client not initialized")
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("generate content failed: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	return resp.Candidates[0].Content, nil
}

// ChatModel creates a chat model client
func ChatModel(ctx context.Context, modelName string) (*GenaiClient, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model or agent disabled")
	}

	m := GetModel(modelName)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: m.ApiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &GenaiClient{
		client:   client,
		model:    modelName,
		provider: m.Provider,
		baseURL:  m.BaseUrl,
		apiKey:   m.ApiKey,
		timeout:  10 * time.Minute,
	}, nil
}

// Generate generates an LLM response
func Generate(ctx context.Context, client *GenaiClient, messages []*Message) (*Message, error) {
	_, err := CountMessageTokens(messages)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	contents := make([]*genai.Content, len(messages))
	for i, msg := range messages {
		contents[i] = msg.ToGenaiContent()
	}

	resp, err := client.Generate(ctx, contents)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}

	return MessageFromGenaiContent(resp), nil
}

// Stream streams an LLM response
func Stream(ctx context.Context, client *GenaiClient, messages []*Message) (<-chan *Message, error) {
	_, err := CountMessageTokens(messages)
	if err != nil {
		return nil, fmt.Errorf("count token failed: %w", err)
	}

	contents := make([]*genai.Content, len(messages))
	for i, msg := range messages {
		contents[i] = msg.ToGenaiContent()
	}

	streamChan := make(chan *Message)
	go func() {
		defer close(streamChan)
		resp, err := client.Generate(ctx, contents)
		if err != nil {
			return
		}
		streamChan <- MessageFromGenaiContent(resp)
	}()

	return streamChan, nil
}
