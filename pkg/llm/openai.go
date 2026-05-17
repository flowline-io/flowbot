package llm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/config"
)

func init() {
	register(ProviderOpenAI, newOpenAIProvider)
	register(ProviderOpenAICompatible, newOpenAIProvider)
}

func newOpenAIProvider(cfg config.Model) (Provider, error) {
	baseURL := "https://api.openai.com/v1"
	if cfg.Provider == ProviderOpenAICompatible && cfg.BaseUrl != "" {
		baseURL = cfg.BaseUrl
	}
	return &openAIProvider{
		apiKey:  cfg.ApiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}, nil
}

// openAIProvider implements Provider for OpenAI and OpenAI-compatible APIs
type openAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func (p *openAIProvider) Name() string { return ProviderOpenAI }

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type openAITool struct {
	Type     string            `json:"type"`
	Function openAIToolFunc    `json:"function"`
}

type openAIToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Tools       []openAITool        `json:"tools,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message      openAIChatMessage  `json:"message,omitempty"`
	Delta        openAIChatDelta    `json:"delta,omitempty"`
	FinishReason string             `json:"finish_reason,omitempty"`
}

type openAIChatDelta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type openAIStreamChunk struct {
	Choices []openAIChoice `json:"choices"`
}

func (p *openAIProvider) Generate(ctx context.Context, req *GenerateRequest) (*Message, error) {
	msgs := make([]openAIChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m == nil {
			continue
		}
		msgs = append(msgs, openAIChatMessage{Role: m.Role, Content: m.Content, Name: m.Name})
	}

	body := openAIChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}
	if len(req.Tools) > 0 {
		body.Tools = convertToolsToOpenAI(req.Tools)
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return &Message{}, nil
	}

	choice := resp.Choices[0]
	return &Message{
		Role:    choice.Message.Role,
		Content: choice.Message.Content,
	}, nil
}

func (p *openAIProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan MessageFragment, error) {
	msgs := make([]openAIChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m == nil {
			continue
		}
		msgs = append(msgs, openAIChatMessage{Role: m.Role, Content: m.Content, Name: m.Name})
	}

	body := openAIChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}
	if len(req.Tools) > 0 {
		body.Tools = convertToolsToOpenAI(req.Tools)
	}

	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("openai stream api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan MessageFragment, 16)
	go p.readSSEStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *openAIProvider) doRequest(ctx context.Context, body openAIChatRequest) (*openAIChatResponse, error) {
	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var result openAIChatResponse
	if err := sonic.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("openai api error: %s", result.Error.Message)
	}

	return &result, nil
}

func (p *openAIProvider) readSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- MessageFragment) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- MessageFragment{Err: context.Cause(ctx)}
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			ch <- MessageFragment{Done: true}
			return
		}

		var chunk openAIStreamChunk
		if err := sonic.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- MessageFragment{Err: fmt.Errorf("parse stream chunk: %w", err)}
			return
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			ch <- MessageFragment{Content: chunk.Choices[0].Delta.Content}
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != "" {
			ch <- MessageFragment{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- MessageFragment{Err: fmt.Errorf("stream read: %w", err)}
		return
	}

	ch <- MessageFragment{Done: true}
}

func convertToolsToOpenAI(tools []Tool) []openAITool {
	result := make([]openAITool, 0, len(tools))
	for _, t := range tools {
		schema, err := t.Parameters().ToJSONSchema()
		if err != nil {
			continue
		}
		result = append(result, openAITool{
			Type: "function",
			Function: openAIToolFunc{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  schema,
			},
		})
	}
	return result
}
