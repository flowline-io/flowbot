package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	agentresult "github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/tmc/langchaingo/llms"
	"go.opentelemetry.io/otel/attribute"
)

// LoopDeps holds runtime dependencies for the agent loop.
type LoopDeps struct {
	Model    llms.Model
	Registry *tool.Registry
}

// RunLoop starts a new agent loop from prompt messages.
func RunLoop(ctx context.Context, prompts []AgentMessage, agentCtx *Context, cfg Config, deps LoopDeps, stream *agentevent.Stream) ([]AgentMessage, error) {
	cfg = cfg.WithDefaults()
	cfg = model.ApplyDefaultRouter(cfg)
	if deps.Registry == nil {
		deps.Registry = tool.NewRegistry()
	}
	if cfg.ConvertToLLM == nil {
		cfg.ConvertToLLM = transform.DefaultConvertToLLM
	}
	if cfg.TransformContext == nil {
		cfg.TransformContext = transform.FilterContext
	}

	newMessages := append([]AgentMessage(nil), prompts...)
	current := cloneContext(agentCtx)
	current.Messages = append(current.Messages, prompts...)

	emit := func(ev agentevent.Event) error {
		if stream == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return ErrAborted
		}
		if err := stream.Push(ctx, ev); err != nil {
			return abortLoopError(err)
		}
		return nil
	}

	if ctx.Err() != nil {
		return newMessages, ErrAborted
	}

	if err := emit(agentevent.Event{Type: agentevent.TypeAgentStart}); err != nil {
		return newMessages, err
	}

	err := runLoopCore(ctx, current, cfg, deps, emit, &newMessages, false)
	if err != nil {
		_ = emit(agentevent.Event{Type: agentevent.TypeAgentEnd, Messages: toAgentMessages(newMessages), Err: err})
		return newMessages, err
	}
	_ = emit(agentevent.Event{Type: agentevent.TypeAgentEnd, Messages: toAgentMessages(newMessages)})
	return newMessages, nil
}

// RunLoopContinue resumes a loop from existing context without adding prompts.
func RunLoopContinue(ctx context.Context, agentCtx *Context, cfg Config, deps LoopDeps, stream *agentevent.Stream) ([]AgentMessage, error) {
	cfg = cfg.WithDefaults()
	cfg = model.ApplyDefaultRouter(cfg)
	if agentCtx == nil || len(agentCtx.Messages) == 0 {
		return nil, ErrEmptyContext
	}
	last := agentCtx.Messages[len(agentCtx.Messages)-1]
	if _, ok := last.(AssistantMessage); ok {
		return nil, ErrInvalidContinue
	}
	if deps.Registry == nil {
		deps.Registry = tool.NewRegistry()
	}
	if cfg.ConvertToLLM == nil {
		cfg.ConvertToLLM = transform.DefaultConvertToLLM
	}
	if cfg.TransformContext == nil {
		cfg.TransformContext = transform.FilterContext
	}

	current := cloneContext(agentCtx)
	var newMessages []AgentMessage
	emit := func(ev agentevent.Event) error {
		if stream == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return ErrAborted
		}
		if err := stream.Push(ctx, ev); err != nil {
			return abortLoopError(err)
		}
		return nil
	}
	if ctx.Err() != nil {
		return nil, ErrAborted
	}
	if err := emit(agentevent.Event{Type: agentevent.TypeAgentStart}); err != nil {
		return nil, err
	}
	err := runLoopCore(ctx, current, cfg, deps, emit, &newMessages, true)
	if err != nil {
		_ = emit(agentevent.Event{Type: agentevent.TypeAgentEnd, Messages: toAgentMessages(newMessages), Err: err})
		return newMessages, err
	}
	_ = emit(agentevent.Event{Type: agentevent.TypeAgentEnd, Messages: toAgentMessages(newMessages)})
	return newMessages, nil
}

func runLoopCore(
	ctx context.Context,
	current *Context,
	cfg Config,
	deps LoopDeps,
	emit func(agentevent.Event) error,
	newMessages *[]AgentMessage,
	continuing bool,
) error {
	steps := 0
	pending := []AgentMessage(nil)
	state := innerLoopState{
		ctx:         ctx,
		current:     current,
		cfg:         cfg,
		deps:        deps,
		emit:        emit,
		newMessages: newMessages,
		pending:     &pending,
		steps:       &steps,
	}

	for {
		if ctx.Err() != nil {
			return ErrAborted
		}

		for {
			stopInner, err := state.runTurn()
			if err == errStopAfterTurn {
				return nil
			}
			if err != nil {
				return err
			}
			if stopInner {
				break
			}
		}

		if cfg.GetFollowUpMessages != nil {
			followUps, followErr := cfg.GetFollowUpMessages()
			if followErr != nil {
				return followErr
			}
			pending = drainQueue(nil, followUps, cfg.FollowUpMode)
			if len(pending) > 0 {
				continue
			}
		}
		break
	}

	_ = continuing
	return nil
}

func streamAssistant(
	ctx context.Context,
	current *Context,
	cfg Config,
	deps LoopDeps,
	emit func(agentevent.Event) error,
) (AssistantMessage, error) {
	ctx, span := trace.StartSpan(ctx, "agent.llm.stream")
	defer span.End()

	messages := current.Messages
	if cfg.TransformContext != nil {
		transformed, err := cfg.TransformContext(messages)
		if err != nil {
			if abortErr := abortLoopError(err); abortErr != err {
				return AssistantMessage{}, abortErr
			}
			return AssistantMessage{}, fmt.Errorf("agent loop: transform context: %w", err)
		}
		messages = transformed
	}

	llmMessages, err := cfg.ConvertToLLM(messages)
	if err != nil {
		return AssistantMessage{}, fmt.Errorf("agent loop: convert to llm: %w", err)
	}

	modelName := turnModelName(cfg, current)
	trace.SetSpanAttributes(ctx, attribute.String("model", modelName))

	activeTools := deps.Registry.ActiveTools()
	llmTools := tool.BuildLLMTools(activeTools)

	if emit != nil {
		if err := emit(agentevent.Event{Type: agentevent.TypeMessageStart, Message: AssistantMessage{}}); err != nil {
			return AssistantMessage{}, err
		}
	}

	retry := llmRetryFromConfig(cfg)
	retry.OnRetry = func(_ int, _ time.Duration, _ error) {
		metrics.Agent().IncLLMRetry(modelName)
	}

	streamOpts := agentllm.StreamOptions{
		ModelName:   modelName,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		Tools:       llmTools,
		OnTextDelta: messageTextDeltaHandler(emit),
		Retry:       retry,
	}
	if agentllm.SupportsReasoningStream(modelName) {
		streamOpts.OnReasoningDelta = messageReasoningDeltaHandler(emit)
	}

	start := time.Now()
	result, err := agentllm.StreamAssistant(ctx, deps.Model, current.SystemPrompt, llmMessages, streamOpts)
	metrics.Agent().ObserveLLMDuration(modelName, time.Since(start).Seconds())
	if err != nil {
		err = agentresult.WrapOverflowError(err)
		metrics.Agent().IncLLMRequest(modelName, "error")
		trace.RecordError(ctx, err)
		return AssistantMessage{}, err
	}
	metrics.Agent().IncLLMRequest(modelName, "ok")

	parts := make([]ContentPart, 0, 1+len(result.ToolCalls))
	if trimmed := msg.TrimToolCallStreamContent(result.Content); trimmed != "" {
		parts = append(parts, TextPart{Text: trimmed})
	}
	for _, call := range result.ToolCalls {
		args := ""
		name := ""
		if call.FunctionCall != nil {
			args = call.FunctionCall.Arguments
			name = call.FunctionCall.Name
		}
		parts = append(parts, ToolCallPart{
			ID:        call.ID,
			Name:      name,
			Arguments: args,
		})
	}

	assistant := AssistantMessage{
		Parts:      parts,
		Model:      result.ModelName,
		StopReason: result.StopReason,
		Usage:      usageToMsg(result.Usage),
	}

	if emit != nil {
		if err := emit(agentevent.Event{Type: agentevent.TypeMessageEnd, Message: assistant}); err != nil {
			return AssistantMessage{}, err
		}
	}

	return assistant, nil
}

func messageTextDeltaHandler(emit func(agentevent.Event) error) func(string) error {
	return func(delta string) error {
		if emit == nil || delta == "" {
			return nil
		}
		return emit(agentevent.Event{Type: agentevent.TypeMessageUpdate, TextDelta: delta})
	}
}

func llmRetryFromConfig(cfg Config) agentllm.RetryConfig {
	retry := agentllm.DefaultRetryConfig()
	if cfg.LLMRetryMaxAttempts > 0 {
		retry.MaxAttempts = cfg.LLMRetryMaxAttempts
	}
	if cfg.LLMRetryInitialInterval > 0 {
		retry.InitialInterval = cfg.LLMRetryInitialInterval
	}
	if cfg.LLMRetryMaxInterval > 0 {
		retry.MaxInterval = cfg.LLMRetryMaxInterval
	}
	if cfg.LLMRetryMultiplier > 0 {
		retry.Multiplier = cfg.LLMRetryMultiplier
	}
	return retry
}

func messageReasoningDeltaHandler(emit func(agentevent.Event) error) func(string) error {
	return func(delta string) error {
		if emit == nil || delta == "" {
			return nil
		}
		return emit(agentevent.Event{Type: agentevent.TypeMessageUpdate, ReasoningDelta: delta})
	}
}

func usageToMsg(usage *agentllm.Usage) *Usage {
	if usage == nil {
		return nil
	}
	return &Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CacheRead:        usage.CacheRead,
		CacheWrite:       usage.CacheWrite,
	}
}

func emitMessage(emit func(agentevent.Event) error, message AgentMessage) error {
	if err := emit(agentevent.Event{Type: agentevent.TypeMessageStart, Message: message}); err != nil {
		return err
	}
	return emit(agentevent.Event{Type: agentevent.TypeMessageEnd, Message: message})
}

func drainQueue(existing, incoming []AgentMessage, mode QueueMode) []AgentMessage {
	combined := make([]AgentMessage, 0, len(existing)+len(incoming))
	combined = append(combined, existing...)
	combined = append(combined, incoming...)
	if mode == QueueOne && len(combined) > 1 {
		return combined[:1]
	}
	return combined
}

func cloneContext(src *Context) *Context {
	if src == nil {
		return &Context{}
	}
	clone := *src
	clone.Messages = append([]AgentMessage(nil), src.Messages...)
	return &clone
}

func toAgentMessages(messages []AgentMessage) []AgentMessage {
	return append([]AgentMessage(nil), messages...)
}

func abortLoopError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ErrAborted
	}
	return err
}

// turnModelName returns the provider model name for the next LLM request.
func turnModelName(cfg Config, current *Context) string {
	if cfg.ModelName != "" {
		return cfg.ModelName
	}
	if current != nil {
		return current.ModelName
	}
	return ""
}
