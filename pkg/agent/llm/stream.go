package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"
)

// ErrAborted indicates the LLM call was cancelled.
var ErrAborted = errors.New("agent llm: aborted")

// StreamOptions configures a streaming assistant request.
type StreamOptions struct {
	ModelName        string
	Temperature      float64
	MaxTokens        int
	Tools            []llms.Tool
	OnTextDelta      func(delta string) error
	OnReasoningDelta func(delta string) error
	// Retry overrides the default transient retry policy when non-zero MaxAttempts is set.
	Retry RetryConfig
}

// AssistantResult is the normalized output of a streaming assistant request.
type AssistantResult struct {
	Content    string
	ToolCalls  []llms.ToolCall
	ModelName  string
	StopReason string
	Usage      *Usage
}

// Usage captures token consumption from an LLM response.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CacheRead        int
	CacheWrite       int
}

// StreamAssistant performs a streaming LLM call and assembles the assistant result.
// Transient failures are retried only before any streaming delta is delivered.
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

	if systemPrompt != "" {
		messages = append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt)}, messages...)
	}

	var result AssistantResult
	retryCfg := opts.Retry
	if retryCfg.MaxAttempts <= 0 {
		retryCfg = DefaultRetryConfig()
	}
	_, err := backoff.Do(ctx, retryCfg.toBackoff(), func(attemptCtx context.Context) error {
		out, callErr := streamAssistantOnce(attemptCtx, model, messages, opts)
		if callErr != nil {
			return callErr
		}
		result = out
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrAborted) || ctx.Err() != nil {
			return AssistantResult{}, ErrAborted
		}
		return AssistantResult{}, err
	}
	return result, nil
}

func streamAssistantOnce(
	ctx context.Context,
	model llms.Model,
	messages []llms.MessageContent,
	opts StreamOptions,
) (AssistantResult, error) {
	callOpts := buildGenerateCallOptions(opts)
	var textBuilder strings.Builder
	var textMu sync.Mutex
	tracker := &streamStartTracker{}
	wrapped := wrapStreamCallbacks(opts, tracker)
	callOpts = append(callOpts, buildAssistantStreamOptions(wrapped, &textBuilder, &textMu)...)

	resp, err := model.GenerateContent(ctx, messages, callOpts...)
	if err != nil {
		return mapGenerateError(ctx, err, tracker.hasStarted())
	}
	return assembleAssistantResult(opts.ModelName, resp, &textBuilder)
}

type streamStartTracker struct {
	mu      sync.Mutex
	started bool
}

func (t *streamStartTracker) mark() {
	t.mu.Lock()
	t.started = true
	t.mu.Unlock()
}

func (t *streamStartTracker) hasStarted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.started
}

func buildGenerateCallOptions(opts StreamOptions) []llms.CallOption {
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
	return callOpts
}

func wrapStreamCallbacks(opts StreamOptions, tracker *streamStartTracker) StreamOptions {
	wrapped := opts
	if opts.OnTextDelta != nil {
		inner := opts.OnTextDelta
		wrapped.OnTextDelta = func(delta string) error {
			if delta != "" {
				tracker.mark()
			}
			return inner(delta)
		}
	}
	if opts.OnReasoningDelta != nil {
		inner := opts.OnReasoningDelta
		wrapped.OnReasoningDelta = func(delta string) error {
			if delta != "" {
				tracker.mark()
			}
			return inner(delta)
		}
	}
	return wrapped
}

func mapGenerateError(ctx context.Context, err error, streamStarted bool) (AssistantResult, error) {
	if ctx.Err() != nil {
		return AssistantResult{}, ErrAborted
	}
	wrappedErr := fmt.Errorf("agent llm: generate content: %w", err)
	if streamStarted {
		return AssistantResult{}, streamStartedError{cause: wrappedErr}
	}
	return AssistantResult{}, wrappedErr
}

func assembleAssistantResult(modelName string, resp *llms.ContentResponse, textBuilder *strings.Builder) (AssistantResult, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return AssistantResult{}, fmt.Errorf("agent llm: empty response")
	}
	choice := resp.Choices[0]
	content := choice.Content
	if content == "" && textBuilder.Len() > 0 {
		content = textBuilder.String()
	}
	if len(choice.ToolCalls) > 0 {
		content = msg.TrimToolCallStreamContent(content)
	}
	stopReason := "complete"
	if choice.StopReason == "tool_calls" || len(choice.ToolCalls) > 0 {
		stopReason = "tool_calls"
	}
	return AssistantResult{
		Content:    content,
		ToolCalls:  append([]llms.ToolCall(nil), choice.ToolCalls...),
		ModelName:  modelName,
		StopReason: stopReason,
		Usage:      usageFromGenerationInfo(choice.GenerationInfo),
	}, nil
}

func buildAssistantStreamOptions(opts StreamOptions, textBuilder *strings.Builder, textMu *sync.Mutex) []llms.CallOption {
	streamText := func(streamCtx context.Context, chunk []byte) error {
		if streamCtx.Err() != nil {
			return streamCtx.Err()
		}
		if len(chunk) == 0 || opts.OnTextDelta == nil {
			return nil
		}
		delta := string(chunk)
		if msg.IsToolCallStreamDelta(delta) {
			return nil
		}
		textMu.Lock()
		_, _ = textBuilder.WriteString(delta)
		textMu.Unlock()
		return opts.OnTextDelta(delta)
	}

	if opts.OnReasoningDelta != nil {
		out := ReasoningCallOptions(opts.ModelName, opts.MaxTokens)
		out = append(out, llms.WithStreamingReasoningFunc(func(streamCtx context.Context, reasoningChunk, chunk []byte) error {
			if streamCtx.Err() != nil {
				return streamCtx.Err()
			}
			if len(reasoningChunk) > 0 {
				if err := opts.OnReasoningDelta(string(reasoningChunk)); err != nil {
					return err
				}
			}
			return streamText(streamCtx, chunk)
		}))
		return out
	}
	if opts.OnTextDelta == nil {
		return nil
	}
	return []llms.CallOption{llms.WithStreamingFunc(streamText)}
}

func usageFromGenerationInfo(info map[string]any) *Usage {
	if len(info) == 0 {
		return nil
	}
	usage := &Usage{}
	if v, ok := intFromInfo(info, "PromptTokens"); ok {
		usage.PromptTokens = v
	}
	if v, ok := intFromInfo(info, "CompletionTokens"); ok {
		usage.CompletionTokens = v
	}
	if v, ok := intFromInfo(info, "TotalTokens"); ok {
		usage.TotalTokens = v
	}
	if usage.TotalTokens == 0 && (usage.PromptTokens > 0 || usage.CompletionTokens > 0) {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return usage
}

func intFromInfo(info map[string]any, key string) (int, bool) {
	raw, ok := info[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// Complete performs a non-streaming completion for auxiliary tasks such as summarization.
// Retry policy follows chat_agent.llm_retry when configured.
func Complete(
	ctx context.Context,
	model llms.Model,
	systemPrompt string,
	messages []llms.MessageContent,
	modelName string,
	maxTokens int,
) (string, error) {
	return CompleteWithRetry(ctx, model, systemPrompt, messages, modelName, maxTokens, RetryConfigFromChatAgent(config.App.ChatAgent.LLMRetry))
}

// CompleteWithRetry performs a non-streaming completion with an explicit retry policy.
func CompleteWithRetry(
	ctx context.Context,
	model llms.Model,
	systemPrompt string,
	messages []llms.MessageContent,
	modelName string,
	maxTokens int,
	retryCfg RetryConfig,
) (string, error) {
	if systemPrompt != "" {
		messages = append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt)}, messages...)
	}
	var content string
	_, err := backoff.Do(ctx, retryCfg.toBackoff(), func(attemptCtx context.Context) error {
		opts := []llms.CallOption{llms.WithModel(modelName)}
		if maxTokens > 0 {
			opts = append(opts, llms.WithMaxTokens(maxTokens))
		}
		resp, callErr := model.GenerateContent(attemptCtx, messages, opts...)
		if callErr != nil {
			if attemptCtx.Err() != nil {
				return ErrAborted
			}
			return fmt.Errorf("agent llm: complete: %w", callErr)
		}
		if resp == nil || len(resp.Choices) == 0 {
			return fmt.Errorf("agent llm: empty completion")
		}
		content = resp.Choices[0].Content
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrAborted) || ctx.Err() != nil {
			return "", ErrAborted
		}
		return "", err
	}
	return content, nil
}
