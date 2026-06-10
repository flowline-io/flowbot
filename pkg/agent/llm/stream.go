package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// ErrAborted indicates the LLM call was cancelled.
var ErrAborted = errors.New("agent llm: aborted")

// StreamOptions configures a streaming assistant request.
type StreamOptions struct {
	ModelName   string
	Temperature float64
	MaxTokens   int
	Tools       []llms.Tool
	OnTextDelta func(delta string) error
}

// AssistantResult is the normalized output of a streaming assistant request.
type AssistantResult struct {
	Content    string
	ToolCalls  []llms.ToolCall
	ModelName  string
	StopReason string
}

// StreamAssistant performs a streaming LLM call and assembles the assistant result.
func StreamAssistant(
	ctx context.Context,
	model llms.Model,
	systemPrompt string,
	messages []llms.MessageContent,
	opts StreamOptions,
) (AssistantResult, error) {
	if ctx.Err() != nil {
		return AssistantResult{}, ErrAborted
	}

	callOpts := []llms.CallOption{
		llms.WithModel(opts.ModelName),
		llms.WithTools(opts.Tools),
	}
	if opts.Temperature > 0 {
		callOpts = append(callOpts, llms.WithTemperature(opts.Temperature))
	}
	if opts.MaxTokens > 0 {
		callOpts = append(callOpts, llms.WithMaxTokens(opts.MaxTokens))
	}

	var textBuilder strings.Builder
	var textMu sync.Mutex

	if opts.OnTextDelta != nil {
		callOpts = append(callOpts, llms.WithStreamingFunc(func(streamCtx context.Context, chunk []byte) error {
			if streamCtx.Err() != nil {
				return streamCtx.Err()
			}
			delta := string(chunk)
			textMu.Lock()
			_, _ = textBuilder.WriteString(delta)
			textMu.Unlock()
			return opts.OnTextDelta(delta)
		}))
	}

	if systemPrompt != "" {
		messages = append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt)}, messages...)
	}

	resp, err := model.GenerateContent(ctx, messages, callOpts...)
	if err != nil {
		if ctx.Err() != nil {
			return AssistantResult{}, ErrAborted
		}
		return AssistantResult{}, fmt.Errorf("agent llm: generate content: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return AssistantResult{}, fmt.Errorf("agent llm: empty response")
	}

	choice := resp.Choices[0]
	content := choice.Content
	if content == "" && textBuilder.Len() > 0 {
		content = textBuilder.String()
	}

	stopReason := "complete"
	if choice.StopReason == "tool_calls" || len(choice.ToolCalls) > 0 {
		stopReason = "tool_calls"
	}

	return AssistantResult{
		Content:    content,
		ToolCalls:  append([]llms.ToolCall(nil), choice.ToolCalls...),
		ModelName:  opts.ModelName,
		StopReason: stopReason,
	}, nil
}

// Complete performs a non-streaming completion for auxiliary tasks such as summarization.
func Complete(
	ctx context.Context,
	model llms.Model,
	systemPrompt string,
	messages []llms.MessageContent,
	modelName string,
) (string, error) {
	if systemPrompt != "" {
		messages = append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt)}, messages...)
	}
	resp, err := model.GenerateContent(ctx, messages, llms.WithModel(modelName))
	if err != nil {
		return "", fmt.Errorf("agent llm: complete: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("agent llm: empty completion")
	}
	return resp.Choices[0].Content, nil
}
