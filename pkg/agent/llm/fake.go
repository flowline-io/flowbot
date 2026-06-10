package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// ResponseScript describes one scripted model response for tests.
type ResponseScript struct {
	Content   string
	ToolCalls []llms.ToolCall
	Err       error
}

// FakeModel implements llms.Model with a queue of scripted responses.
type FakeModel struct {
	mu        sync.Mutex
	responses []ResponseScript
	calls     int
}

// NewFakeModel creates a fake model with the given response sequence.
func NewFakeModel(responses ...ResponseScript) *FakeModel {
	return &FakeModel{responses: append([]ResponseScript(nil), responses...)}
}

// Calls returns how many GenerateContent invocations have occurred.
func (f *FakeModel) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Call implements the deprecated llms.Model Call method.
func (f *FakeModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	resp, err := f.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, prompt)}, options...)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("fake model: empty response")
	}
	return resp.Choices[0].Content, nil
}

// GenerateContent returns the next scripted response.
func (f *FakeModel) GenerateContent(ctx context.Context, _ []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++

	if len(f.responses) == 0 {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{{Content: "done", StopReason: "stop"}},
		}, nil
	}

	script := f.responses[0]
	f.responses = f.responses[1:]
	if script.Err != nil {
		return nil, script.Err
	}

	if opts.StreamingFunc != nil && script.Content != "" {
		if err := opts.StreamingFunc(ctx, []byte(script.Content)); err != nil {
			return nil, err
		}
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content:    script.Content,
				ToolCalls:  append([]llms.ToolCall(nil), script.ToolCalls...),
				StopReason: "stop",
			},
		},
	}, nil
}
