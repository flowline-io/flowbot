package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// ResponseScript describes one scripted model response for tests.
type ResponseScript struct {
	Content string
	// Chunks, when non-empty, are emitted one-by-one via StreamingFunc instead of Content as a single chunk.
	Chunks []string
	// ReasoningChunks are emitted via StreamingReasoningFunc when configured.
	ReasoningChunks []string
	ToolCalls       []llms.ToolCall
	Err             error
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

	content := script.Content
	if len(script.Chunks) > 0 {
		content = joinChunks(script.Chunks, script.Content)
	}

	if err := emitFakeStreaming(ctx, opts, script); err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content:    content,
				ToolCalls:  append([]llms.ToolCall(nil), script.ToolCalls...),
				StopReason: "stop",
			},
		},
	}, nil
}

func joinChunks(chunks []string, fallback string) string {
	if fallback != "" {
		return fallback
	}
	return strings.Join(chunks, "")
}

func emitFakeStreaming(ctx context.Context, opts llms.CallOptions, script ResponseScript) error {
	if opts.StreamingReasoningFunc != nil {
		return emitReasoningStreaming(ctx, opts.StreamingReasoningFunc, script)
	}
	if opts.StreamingFunc != nil {
		return emitTextStreaming(ctx, opts.StreamingFunc, script)
	}
	return nil
}

func emitReasoningStreaming(
	ctx context.Context,
	stream func(context.Context, []byte, []byte) error,
	script ResponseScript,
) error {
	for _, chunk := range script.ReasoningChunks {
		if err := stream(ctx, []byte(chunk), nil); err != nil {
			return err
		}
	}
	chunks := script.Chunks
	if len(chunks) == 0 && script.Content != "" {
		return stream(ctx, nil, []byte(script.Content))
	}
	for _, chunk := range chunks {
		if err := stream(ctx, nil, []byte(chunk)); err != nil {
			return err
		}
	}
	return nil
}

func emitTextStreaming(ctx context.Context, stream func(context.Context, []byte) error, script ResponseScript) error {
	chunks := script.Chunks
	if len(chunks) == 0 {
		if script.Content == "" {
			return nil
		}
		return stream(ctx, []byte(script.Content))
	}
	for _, chunk := range chunks {
		if err := stream(ctx, []byte(chunk)); err != nil {
			return err
		}
	}
	return nil
}
