package ability

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Invoker func(ctx context.Context, params map[string]any) (*InvokeResult, error)

type Registry struct {
	mu       sync.RWMutex
	handlers map[hub.CapabilityType]map[string]Invoker
	emitter  EventEmitter
}

type EventEmitter func(ctx context.Context, result *InvokeResult)

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[hub.CapabilityType]map[string]Invoker)}
}

func SetEventEmitter(emitter EventEmitter) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	DefaultRegistry.emitter = emitter
}

func RegisterInvoker(capability hub.CapabilityType, operation string, invoker Invoker) error {
	return DefaultRegistry.Register(capability, operation, invoker)
}

func Invoke(ctx context.Context, capability hub.CapabilityType, operation string, params map[string]any) (*InvokeResult, error) {
	return DefaultRegistry.Invoke(ctx, capability, operation, params)
}

func (r *Registry) Register(capability hub.CapabilityType, operation string, invoker Invoker) error {
	if capability == "" {
		return types.Errorf(types.ErrInvalidArgument, "capability is required")
	}
	if operation == "" {
		return types.Errorf(types.ErrInvalidArgument, "operation is required")
	}
	if invoker == nil {
		return types.Errorf(types.ErrInvalidArgument, "invoker is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers[capability] == nil {
		r.handlers[capability] = make(map[string]Invoker)
	}
	r.handlers[capability][operation] = invoker
	return nil
}

func (r *Registry) Invoke(ctx context.Context, capability hub.CapabilityType, operation string, params map[string]any) (*InvokeResult, error) {
	if params == nil {
		params = map[string]any{}
	}
	r.mu.RLock()
	ops, ok := r.handlers[capability]
	if !ok {
		r.mu.RUnlock()
		return nil, types.Errorf(types.ErrNotFound, "capability %s not found", capability)
	}
	invoker, ok := ops[operation]
	r.mu.RUnlock()
	if !ok {
		return nil, types.Errorf(types.ErrNotImplemented, "operation %s.%s not implemented", capability, operation)
	}
	result, err := invoker(ctx, params)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &InvokeResult{}
	}
	result.Capability = capability
	result.Operation = operation

	r.mu.RLock()
	emitter := r.emitter
	r.mu.RUnlock()
	if emitter != nil && len(result.Events) > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					flog.Warn("ability(%s.%s): event emitter panicked: %v", capability, operation, r)
				}
			}()
			emitter(context.WithoutCancel(ctx), result)
		}()
	}

	return result, nil
}
