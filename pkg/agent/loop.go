package agent

import (
	"context"
	"fmt"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/tmc/langchaingo/llms"
)

// LoopDeps holds runtime dependencies for the agent loop.
type LoopDeps struct {
	Model    llms.Model
	Registry *tool.Registry
}

// RunLoop starts a new agent loop from prompt messages.
func RunLoop(ctx context.Context, prompts []AgentMessage, agentCtx *Context, cfg Config, deps LoopDeps, stream *agentevent.Stream) ([]AgentMessage, error) {
	cfg = cfg.WithDefaults()
	cfg = applyDefaultRouter(cfg)
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
		return stream.Push(ctx, ev)
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
	cfg = applyDefaultRouter(cfg)
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
		return stream.Push(ctx, ev)
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
	messages := current.Messages
	if cfg.TransformContext != nil {
		transformed, err := cfg.TransformContext(messages)
		if err != nil {
			return AssistantMessage{}, fmt.Errorf("agent loop: transform context: %w", err)
		}
		messages = transformed
	}

	llmMessages, err := cfg.ConvertToLLM(messages)
	if err != nil {
		return AssistantMessage{}, fmt.Errorf("agent loop: convert to llm: %w", err)
	}

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = current.ModelName
	}

	activeTools := deps.Registry.ActiveTools()
	llmTools := tool.BuildLLMTools(activeTools)

	if emit != nil {
		if err := emit(agentevent.Event{Type: agentevent.TypeMessageStart, Message: AssistantMessage{}}); err != nil {
			return AssistantMessage{}, err
		}
	}

	result, err := agentllm.StreamAssistant(ctx, deps.Model, current.SystemPrompt, llmMessages, agentllm.StreamOptions{
		ModelName:   modelName,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		Tools:       llmTools,
		OnTextDelta: func(delta string) error {
			if emit == nil || delta == "" {
				return nil
			}
			return emit(agentevent.Event{Type: agentevent.TypeMessageUpdate, TextDelta: delta})
		},
	})
	if err != nil {
		return AssistantMessage{}, err
	}

	parts := make([]ContentPart, 0, 1+len(result.ToolCalls))
	if result.Content != "" {
		parts = append(parts, TextPart{Text: result.Content})
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

func applyDefaultRouter(cfg Config) Config {
	if cfg.PrepareNextTurn != nil || cfg.ChatModel == "" || cfg.ToolModel == "" {
		return cfg
	}
	router := model.NewRouter(cfg.ChatModel, cfg.ToolModel)
	cfg.PrepareNextTurn = func(turn TurnContext) (*TurnUpdate, error) {
		ctx := cloneContext(turn.Context)
		router.ApplyToContext(ctx, len(turn.ToolResults) > 0)
		return &TurnUpdate{Context: ctx, ModelName: ctx.ModelName}, nil
	}
	if cfg.ModelName == "" {
		cfg.ModelName = cfg.ChatModel
	}
	return cfg
}
