package ability

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Invoker func(ctx context.Context, params map[string]any) (*InvokeResult, error)

type Registry struct {
	mu       sync.RWMutex
	handlers map[hub.CapabilityType]map[string]Invoker
	emitter  EventEmitter
	metrics  *metrics.AbilityCollector
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

// SetMetricsCollector sets the AbilityCollector on the DefaultRegistry.
func SetMetricsCollector(mc *metrics.AbilityCollector) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	DefaultRegistry.metrics = mc
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

	ctx, span := trace.StartSpan(ctx, "ability."+string(capability)+"."+operation,
		attribute.String("capability.name", string(capability)),
		attribute.String("capability.operation", operation),
	)
	defer span.End()

	start := time.Now()
	result, err := invoker(ctx, params)
	if err != nil {
		trace.RecordError(ctx, err)
		r.mu.RLock()
		mc := r.metrics
		r.mu.RUnlock()
		if mc != nil {
			mc.IncInvokeTotal(string(capability), operation, "error")
			mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
			code := "unknown"
			if te, ok := err.(*types.Error); ok {
				code = te.Code
			}
			mc.IncInvokeError(string(capability), operation, code)
		}
		return nil, err
	}
	if result == nil {
		result = &InvokeResult{}
	}
	result.Capability = capability
	result.Operation = operation

	r.mu.RLock()
	mc := r.metrics
	r.mu.RUnlock()
	if mc != nil {
		mc.IncInvokeTotal(string(capability), operation, "ok")
		mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
	}

	r.mu.RLock()
	emitter := r.emitter
	r.mu.RUnlock()
	if emitter != nil && len(result.Events) > 0 {
		capt := string(capability)
		op := operation
		res := result
		submitEvent(capt, op, func() {
			emitter(context.WithoutCancel(ctx), res)
		})
	}

	return result, nil
}
