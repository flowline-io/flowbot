package hooks

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

type (
	beforeAgentStartHandler      func(context.Context, BeforeAgentStartEvent) (*BeforeAgentStartResult, error)
	contextHandler               func(context.Context, ContextEvent) (*ContextResult, error)
	beforeProviderRequestHandler func(context.Context, BeforeProviderRequestEvent) (*BeforeProviderRequestResult, error)
	toolCallHandler              func(context.Context, ToolCallEvent) (*ToolCallResult, error)
	toolResultHandler            func(context.Context, ToolResultEvent) (*ToolResultResult, error)
	sessionBeforeCompactHandler  func(context.Context, SessionBeforeCompactEvent) (*SessionBeforeCompactResult, error)
	sessionBeforeTreeHandler     func(context.Context, SessionBeforeTreeEvent) (*SessionBeforeTreeResult, error)
	observationHandler           func(context.Context, ObservationEvent) error
)

// Registry stores typed hook handlers and observation listeners.
type Registry struct {
	mu sync.RWMutex

	beforeAgentStart      []beforeAgentStartHandler
	context               []contextHandler
	beforeProviderRequest []beforeProviderRequestHandler
	toolCall              []toolCallHandler
	toolResult            []toolResultHandler
	sessionBeforeCompact  []sessionBeforeCompactHandler
	sessionBeforeTree     []sessionBeforeTreeHandler
	observe               []observationHandler
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
		len(r.beforeProviderRequest) > 0 ||
		len(r.toolCall) > 0 ||
		len(r.toolResult) > 0
}

// HasSessionHandlers reports whether any session compact/tree hook is registered.
func (r *Registry) HasSessionHandlers() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessionBeforeCompact) > 0 || len(r.sessionBeforeTree) > 0
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
		len(r.beforeProviderRequest) > 0 ||
		len(r.toolCall) > 0 ||
		len(r.toolResult) > 0 ||
		len(r.sessionBeforeCompact) > 0 ||
		len(r.sessionBeforeTree) > 0 ||
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

// OnBeforeProviderRequest registers a handler that patches LLM stream options.
func OnBeforeProviderRequest(reg *Registry, handler beforeProviderRequestHandler) {
	if reg == nil {
		return
	}
	reg.registerBeforeProviderRequest(handler)
}

// OnSessionBeforeCompact registers a handler that can cancel or customize compaction.
func OnSessionBeforeCompact(reg *Registry, handler sessionBeforeCompactHandler) {
	if reg == nil {
		return
	}
	reg.registerSessionBeforeCompact(handler)
}

// OnSessionBeforeTree registers a handler that can cancel or customize branch summary.
func OnSessionBeforeTree(reg *Registry, handler sessionBeforeTreeHandler) {
	if reg == nil {
		return
	}
	reg.registerSessionBeforeTree(handler)
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

func (r *Registry) registerBeforeProviderRequest(handler beforeProviderRequestHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beforeProviderRequest = append(r.beforeProviderRequest, handler)
}

func (r *Registry) registerSessionBeforeCompact(handler sessionBeforeCompactHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionBeforeCompact = append(r.sessionBeforeCompact, handler)
}

func (r *Registry) registerSessionBeforeTree(handler sessionBeforeTreeHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionBeforeTree = append(r.sessionBeforeTree, handler)
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

func (r *Registry) handlersBeforeProviderRequest() []beforeProviderRequestHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]beforeProviderRequestHandler(nil), r.beforeProviderRequest...)
}

func (r *Registry) handlersSessionBeforeCompact() []sessionBeforeCompactHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]sessionBeforeCompactHandler(nil), r.sessionBeforeCompact...)
}

func (r *Registry) handlersSessionBeforeTree() []sessionBeforeTreeHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]sessionBeforeTreeHandler(nil), r.sessionBeforeTree...)
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

// EmitBeforeProviderRequest runs before_provider_request handlers and chains option patches.
func (r *Registry) EmitBeforeProviderRequest(ctx context.Context, event BeforeProviderRequestEvent) (*BeforeProviderRequestResult, error) {
	if r == nil {
		return nil, nil
	}
	current := event.Options
	changed := false
	for _, handler := range r.handlersBeforeProviderRequest() {
		result, err := handler(ctx, BeforeProviderRequestEvent{
			ModelName: event.ModelName,
			Options:   current,
		})
		if err != nil {
			return nil, err
		}
		if result != nil && result.Options != nil {
			current = *result.Options
			changed = true
		}
	}
	if !changed {
		return nil, nil
	}
	opts := current
	return &BeforeProviderRequestResult{Options: &opts}, nil
}

// EmitSessionBeforeCompact runs session_before_compact handlers (first cancel, else last compaction).
func (r *Registry) EmitSessionBeforeCompact(ctx context.Context, event SessionBeforeCompactEvent) (*SessionBeforeCompactResult, error) {
	if r == nil {
		return nil, nil
	}
	handlers := r.handlersSessionBeforeCompact()
	cancel, last, err := reduceFirstCancelOrLast(len(handlers), func(i int) (bool, *ctxmgr.CompactionResult, error) {
		result, err := handlers[i](ctx, event)
		if err != nil || result == nil {
			return false, nil, err
		}
		if result.Cancel {
			return true, nil, nil
		}
		return false, result.Compaction, nil
	})
	if err != nil {
		return nil, err
	}
	if cancel {
		return &SessionBeforeCompactResult{Cancel: true}, nil
	}
	if last == nil {
		return nil, nil
	}
	compacted := *last
	return &SessionBeforeCompactResult{Compaction: &compacted}, nil
}

// EmitSessionBeforeTree runs session_before_tree handlers (first cancel, else last summary).
func (r *Registry) EmitSessionBeforeTree(ctx context.Context, event SessionBeforeTreeEvent) (*SessionBeforeTreeResult, error) {
	if r == nil {
		return nil, nil
	}
	handlers := r.handlersSessionBeforeTree()
	cancel, last, err := reduceFirstCancelOrLast(len(handlers), func(i int) (bool, *string, error) {
		result, err := handlers[i](ctx, event)
		if err != nil || result == nil {
			return false, nil, err
		}
		if result.Cancel {
			return true, nil, nil
		}
		return false, result.Summary, nil
	})
	if err != nil {
		return nil, err
	}
	if cancel {
		return &SessionBeforeTreeResult{Cancel: true}, nil
	}
	if last == nil {
		return nil, nil
	}
	summary := *last
	return &SessionBeforeTreeResult{Summary: &summary}, nil
}

// reduceFirstCancelOrLast walks handlers in order: first cancel wins, else last non-nil value.
func reduceFirstCancelOrLast[T any](n int, each func(i int) (cancel bool, value *T, err error)) (bool, *T, error) {
	var last *T
	for i := 0; i < n; i++ {
		cancel, value, err := each(i)
		if err != nil {
			return false, nil, err
		}
		if cancel {
			return true, nil, nil
		}
		if value != nil {
			copied := *value
			last = &copied
		}
	}
	return false, last, nil
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
