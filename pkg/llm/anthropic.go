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

// RegisterAnthropic registers the Anthropic LLM provider in the global provider registry.
func RegisterAnthropic() {
	register(ProviderAnthropic, newAnthropicProvider)
}

const (
	anthropicBaseURL = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
)

func newAnthropicProvider(cfg config.Model) (Provider, error) {
	baseURL := anthropicBaseURL
	if cfg.BaseUrl != "" {
		baseURL = cfg.BaseUrl
	}
	return &anthropicProvider{
		apiKey:  cfg.ApiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}, nil
}

// anthropicProvider implements Provider for Anthropic Claude API
type anthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func (*anthropicProvider) Name() string { return ProviderAnthropic }

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Role    string                  `json:"role"`
	Error   *anthropicError         `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Anthropic streaming event types
type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Delta *anthropicDelta `json:"delta,omitempty"`
	Error *anthropicError `json:"error,omitempty"`
}

type anthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (p *anthropicProvider) Generate(ctx context.Context, req *GenerateRequest) (*Message, error) {
	msgs, system := convertMessagesToAnthropic(req.Messages)

	body := anthropicRequest{
		Model:     req.Model,
		Messages:  msgs,
		System:    system,
		MaxTokens: req.MaxTokens,
		Stream:    false,
	}
	if body.MaxTokens <= 0 {
		body.MaxTokens = 4096
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	return anthropicResponseToMessage(resp), nil
}

func (p *anthropicProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan MessageFragment, error) {
	msgs, system := convertMessagesToAnthropic(req.Messages)

	body := anthropicRequest{
		Model:     req.Model,
		Messages:  msgs,
		System:    system,
		MaxTokens: req.MaxTokens,
		Stream:    true,
	}
	if body.MaxTokens <= 0 {
		body.MaxTokens = 4096
	}

	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("anthropic stream api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan MessageFragment, 16)
	go p.readSSEStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *anthropicProvider) doRequest(ctx context.Context, body anthropicRequest) (*anthropicResponse, error) {
	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

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
		return nil, fmt.Errorf("anthropic api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var result anthropicResponse
	if err := sonic.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("anthropic api error: %s", result.Error.Message)
	}

	return &result, nil
}

func (*anthropicProvider) readSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- MessageFragment) {
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

		var event anthropicStreamEvent
		if err := sonic.Unmarshal([]byte(data), &event); err != nil {
			ch <- MessageFragment{Err: fmt.Errorf("parse stream chunk: %w", err)}
			return
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				ch <- MessageFragment{Content: event.Delta.Text}
			}
		case "message_stop":
			ch <- MessageFragment{Done: true}
			return
		case "error":
			if event.Error != nil {
				ch <- MessageFragment{Err: fmt.Errorf("anthropic stream error: %s", event.Error.Message)}
			} else {
				ch <- MessageFragment{Err: fmt.Errorf("anthropic stream error event without details")}
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- MessageFragment{Err: fmt.Errorf("stream read: %w", err)}
		return
	}

	ch <- MessageFragment{Done: true}
}

func convertMessagesToAnthropic(messages []*Message) ([]anthropicMessage, string) {
	var system strings.Builder
	msgs := make([]anthropicMessage, 0, len(messages))

	for _, m := range messages {
		if m == nil {
			continue
		}
		if m.Role == SystemRole {
			_, _ = system.WriteString(m.Content + "\n")
		} else {
			role := m.Role
			if role == AssistantRole {
				role = "assistant"
			}
			msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
		}
	}

	return msgs, strings.TrimSpace(system.String())
}

func anthropicResponseToMessage(resp *anthropicResponse) *Message {
	if resp == nil || len(resp.Content) == 0 {
		return &Message{}
	}
	var text strings.Builder
	for _, c := range resp.Content {
		if c.Type == "text" {
			_, _ = text.WriteString(c.Text)
		}
	}
	return &Message{
		Role:    resp.Role,
		Content: text.String(),
	}
}
