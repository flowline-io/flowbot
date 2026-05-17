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
	register(ProviderGemini, newGeminiProvider)
}

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

func newGeminiProvider(cfg config.Model) (Provider, error) {
	baseURL := geminiBaseURL
	if cfg.BaseUrl != "" {
		baseURL = cfg.BaseUrl
	}
	return &geminiProvider{
		apiKey:  cfg.ApiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}, nil
}

// geminiProvider implements Provider for Google Gemini API
type geminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func (p *geminiProvider) Name() string { return ProviderGemini }

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (p *geminiProvider) Generate(ctx context.Context, req *GenerateRequest) (*Message, error) {
	contents := make([]geminiContent, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m == nil {
			continue
		}
		contents = append(contents, geminiContent{Role: m.Role, Parts: []geminiPart{{Text: m.Content}}})
	}

	body := geminiRequest{Contents: contents}

	resp, err := p.doRequest(ctx, req.Model, body)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 {
		return &Message{}, nil
	}

	candidate := resp.Candidates[0]
	content := joinGeminiParts(candidate.Content.Parts)
	return &Message{
		Role:    candidate.Content.Role,
		Content: content,
	}, nil
}

func (p *geminiProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan MessageFragment, error) {
	contents := make([]geminiContent, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m == nil {
			continue
		}
		contents = append(contents, geminiContent{Role: m.Role, Parts: []geminiPart{{Text: m.Content}}})
	}

	body := geminiRequest{Contents: contents}
	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("gemini stream api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan MessageFragment, 16)
	go p.readSSEStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *geminiProvider) doRequest(ctx context.Context, model string, body geminiRequest) (*geminiResponse, error) {
	jsonBody, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("gemini api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var result geminiResponse
	if err := sonic.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("gemini api error: %s", result.Error.Message)
	}

	return &result, nil
}

func (p *geminiProvider) readSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- MessageFragment) {
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

		var chunk geminiResponse
		if err := sonic.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- MessageFragment{Err: fmt.Errorf("parse stream chunk: %w", err)}
			return
		}

		if len(chunk.Candidates) == 0 {
			continue
		}

		candidate := chunk.Candidates[0]
		text := joinGeminiParts(candidate.Content.Parts)
		if text != "" {
			ch <- MessageFragment{Content: text}
		}

		if candidate.FinishReason != "" {
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

func joinGeminiParts(parts []geminiPart) string {
	var b strings.Builder
	for _, p := range parts {
		_, _ = b.WriteString(p.Text)
	}
	return b.String()
}
