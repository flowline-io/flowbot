package hooks

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

type (
	beforeAgentStartHandler func(context.Context, BeforeAgentStartEvent) (*BeforeAgentStartResult, error)
	contextHandler          func(context.Context, ContextEvent) (*ContextResult, error)
	toolCallHandler         func(context.Context, ToolCallEvent) (*ToolCallResult, error)
	toolResultHandler       func(context.Context, ToolResultEvent) (*ToolResultResult, error)
	observationHandler      func(context.Context, ObservationEvent) error
)

// Registry stores typed hook handlers and observation listeners.
type Registry struct {
	mu sync.RWMutex

	beforeAgentStart []beforeAgentStartHandler
	context          []contextHandler
	toolCall         []toolCallHandler
	toolResult       []toolResultHandler
	observe          []observationHandler
}

// NewRegistry creates an empty hook registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// HasLoopHandlers reports whether any loop-bridging hook handler is registered.
func (r *Registry) HasLoopHandlers() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.beforeAgentStart) > 0 ||
		len(r.context) > 0 ||
		len(r.toolCall) > 0 ||
		len(r.toolResult) > 0
}

// HasHandlers reports whether any hook or observer is registered.
func (r *Registry) HasHandlers() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.beforeAgentStart) > 0 ||
		len(r.context) > 0 ||
		len(r.toolCall) > 0 ||
		len(r.toolResult) > 0 ||
		len(r.observe) > 0
}

// OnBeforeAgentStart registers a handler that can mutate prompts before a run.
func OnBeforeAgentStart(reg *Registry, handler beforeAgentStartHandler) {
	if reg == nil {
		return
	}
	reg.registerBeforeAgentStart(handler)
}

// OnContext registers a handler that transforms messages before each LLM call.
func OnContext(reg *Registry, handler contextHandler) {
	if reg == nil {
		return
	}
	reg.registerContext(handler)
}

// OnToolCall registers a handler that can block tool execution.
func OnToolCall(reg *Registry, handler toolCallHandler) {
	if reg == nil {
		return
	}
	reg.registerToolCall(handler)
}

// OnToolResult registers a handler that can patch tool results.
func OnToolResult(reg *Registry, handler toolResultHandler) {
	if reg == nil {
		return
	}
	reg.registerToolResult(handler)
}

// Observe registers a read-only listener for harness observation events.
func Observe(reg *Registry, handler observationHandler) {
	if reg == nil {
		return
	}
	reg.registerObserve(handler)
}

// OnObservation registers an observer filtered to a single event type.
func OnObservation(reg *Registry, eventType string, handler func(context.Context, ObservationEvent) error) {
	if reg == nil {
		return
	}
	Observe(reg, func(ctx context.Context, event ObservationEvent) error {
		if event.Type != eventType {
			return nil
		}
		return handler(ctx, event)
	})
}

func (r *Registry) registerBeforeAgentStart(handler beforeAgentStartHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beforeAgentStart = append(r.beforeAgentStart, handler)
}

func (r *Registry) registerContext(handler contextHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.context = append(r.context, handler)
}

func (r *Registry) registerToolCall(handler toolCallHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolCall = append(r.toolCall, handler)
}

func (r *Registry) registerToolResult(handler toolResultHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolResult = append(r.toolResult, handler)
}

func (r *Registry) registerObserve(handler observationHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observe = append(r.observe, handler)
}

func (r *Registry) handlersBeforeAgentStart() []beforeAgentStartHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]beforeAgentStartHandler(nil), r.beforeAgentStart...)
}

func (r *Registry) handlersContext() []contextHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]contextHandler(nil), r.context...)
}

func (r *Registry) handlersToolCall() []toolCallHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]toolCallHandler(nil), r.toolCall...)
}

func (r *Registry) handlersToolResult() []toolResultHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]toolResultHandler(nil), r.toolResult...)
}

func (r *Registry) handlersObserve() []observationHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]observationHandler(nil), r.observe...)
}

// EmitBeforeAgentStart runs before_agent_start handlers and merges results.
func (r *Registry) EmitBeforeAgentStart(ctx context.Context, event BeforeAgentStartEvent) (*BeforeAgentStartResult, error) {
	if r == nil {
		return nil, nil
	}
	messages := append([]msg.AgentMessage(nil), event.Messages...)
	systemPrompt := event.SystemPrompt
	var merged BeforeAgentStartResult

	for _, handler := range r.handlersBeforeAgentStart() {
		result, err := handler(ctx, BeforeAgentStartEvent{
			Messages:     append([]msg.AgentMessage(nil), messages...),
			SystemPrompt: systemPrompt,
			ModelName:    event.ModelName,
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		if result.Cancel {
			merged.Cancel = true
		}
		if result.Messages != nil {
			messages = append([]msg.AgentMessage(nil), result.Messages...)
			merged.Messages = messages
		}
		if result.SystemPrompt != nil {
			systemPrompt = *result.SystemPrompt
			prompt := systemPrompt
			merged.SystemPrompt = &prompt
		}
	}

	if !merged.Cancel && merged.Messages == nil && merged.SystemPrompt == nil {
		return nil, nil
	}
	if merged.Messages == nil && len(messages) != len(event.Messages) {
		merged.Messages = messages
	}
	if merged.SystemPrompt == nil && systemPrompt != event.SystemPrompt {
		prompt := systemPrompt
		merged.SystemPrompt = &prompt
	}
	return &merged, nil
}

// EmitContext runs context handlers and chains message replacements.
func (r *Registry) EmitContext(ctx context.Context, messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
	if r == nil {
		return messages, nil
	}
	current := append([]msg.AgentMessage(nil), messages...)
	for _, handler := range r.handlersContext() {
		result, err := handler(ctx, ContextEvent{Messages: current})
		if err != nil {
			return nil, err
		}
		if result != nil && result.Messages != nil {
			current = append([]msg.AgentMessage(nil), result.Messages...)
		}
	}
	return current, nil
}

// EmitToolCall runs tool_call handlers and stops on the first block.
func (r *Registry) EmitToolCall(ctx context.Context, event ToolCallEvent) (*ToolCallResult, error) {
	if r == nil {
		return nil, nil
	}
	for _, handler := range r.handlersToolCall() {
		result, err := handler(ctx, event)
		if err != nil {
			return nil, err
		}
		if result != nil && result.Block {
			return result, nil
		}
	}
	return nil, nil
}

// EmitToolResult runs tool_result handlers and merges patches.
func (r *Registry) EmitToolResult(ctx context.Context, event ToolResultEvent) (*ToolResultResult, error) {
	if r == nil {
		return nil, nil
	}
	var merged ToolResultResult
	for _, handler := range r.handlersToolResult() {
		result, err := handler(ctx, event)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		if len(result.Parts) > 0 {
			merged.Parts = append([]msg.ContentPart(nil), result.Parts...)
			event.Result.Parts = merged.Parts
		}
		if result.IsError != nil {
			value := *result.IsError
			merged.IsError = &value
			event.Result.IsError = value
		}
		if result.Terminate {
			merged.Terminate = true
		}
	}
	if len(merged.Parts) == 0 && merged.IsError == nil && !merged.Terminate {
		return nil, nil
	}
	return &merged, nil
}

// EmitObservation notifies observers and logs handler errors without failing the run.
func (r *Registry) EmitObservation(ctx context.Context, event ObservationEvent, logWarn func(string, ...any)) {
	if r == nil {
		return
	}
	for _, handler := range r.handlersObserve() {
		if err := handler(ctx, event); err != nil && logWarn != nil {
			logWarn("agent hooks: observation %s: %v", event.Type, err)
		}
	}
}
