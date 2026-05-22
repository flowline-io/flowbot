package ability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/bulkhead"
	"github.com/flowline-io/flowbot/pkg/cache"
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

// SetBulkheadCallbacks wires the bulkhead manager with metrics reporting callbacks.
func SetBulkheadCallbacks() {
	bulkhead.SetDefaults(
		bulkhead.WithOnEnter(func(name string, d time.Duration) {
			DefaultRegistry.mu.RLock()
			mc := DefaultRegistry.metrics
			DefaultRegistry.mu.RUnlock()
			if mc != nil {
				mc.IncBulkheadActive(name)
				mc.ObserveBulkheadWaitDuration(name, d.Seconds())
			}
		}),
		bulkhead.WithOnLeave(func(name string) {
			DefaultRegistry.mu.RLock()
			mc := DefaultRegistry.metrics
			DefaultRegistry.mu.RUnlock()
			if mc != nil {
				mc.DecBulkheadActive(name)
			}
		}),
		bulkhead.WithOnDrop(func(name string, reason string) {
			DefaultRegistry.mu.RLock()
			mc := DefaultRegistry.metrics
			DefaultRegistry.mu.RUnlock()
			if mc != nil {
				mc.IncBulkheadDropped(name, reason)
			}
		}),
		bulkhead.WithOnQueueEnter(func(name string) {
			DefaultRegistry.mu.RLock()
			mc := DefaultRegistry.metrics
			DefaultRegistry.mu.RUnlock()
			if mc != nil {
				mc.IncBulkheadQueued(name)
			}
		}),
		bulkhead.WithOnQueueLeave(func(name string) {
			DefaultRegistry.mu.RLock()
			mc := DefaultRegistry.metrics
			DefaultRegistry.mu.RUnlock()
			if mc != nil {
				mc.DecBulkheadQueued(name)
			}
		}),
	)
}

func buildCacheKey(capability hub.CapabilityType, operation string, params map[string]any) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	sorted := make(map[string]any, len(keys))
	for _, k := range keys {
		sorted[k] = params[k]
	}
	data, _ := sonic.MarshalString(sorted)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	hash := hex.EncodeToString(h.Sum(nil))
	return "ability:" + string(capability) + ":" + operation + ":" + hash
}

func hasCursorParam(params map[string]any) bool {
	_, ok := params["cursor"]
	return ok
}

func cacheRead(capability hub.CapabilityType, operation string, params map[string]any) *InvokeResult {
	if cache.Instance == nil {
		return nil
	}
	cacheKey := buildCacheKey(capability, operation, params)
	cached, ok := cache.Instance.GetBytes(cacheKey)
	if !ok {
		return nil
	}
	var result InvokeResult
	if err := sonic.UnmarshalString(string(cached), &result); err != nil {
		return nil
	}
	return &result
}

func cacheWrite(capability hub.CapabilityType, operation string, params map[string]any, result *InvokeResult) {
	if cache.Instance == nil {
		return
	}
	cacheKey := buildCacheKey(capability, operation, params)
	clone := *result
	clone.Events = nil
	data, err := sonic.MarshalString(&clone)
	if err != nil {
		return
	}
	cache.Instance.SetWithTTLCap(cacheKey, []byte(data), 1, cache.TTLShort.Duration(), string(capability))
	cache.Instance.Wait()
}

func cacheInvalidate(capability hub.CapabilityType) {
	if cache.Instance == nil {
		return
	}
	cache.Instance.DelByPrefix(string(capability))
}

func RegisterInvoker(capability hub.CapabilityType, operation string, invoker Invoker) error {
	return DefaultRegistry.Register(capability, operation, invoker)
}

func (r *Registry) recordErrorMetrics(capability hub.CapabilityType, operation string, start time.Time, err error) {
	r.mu.RLock()
	mc := r.metrics
	r.mu.RUnlock()
	if mc == nil {
		return
	}
	mc.IncInvokeTotal(string(capability), operation, "error")
	mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
	code := "unknown"
	if te, ok := err.(*types.Error); ok {
		code = te.Code
	}
	mc.IncInvokeError(string(capability), operation, code)
}

func (r *Registry) recordSuccessMetrics(capability hub.CapabilityType, operation string, start time.Time) {
	r.mu.RLock()
	mc := r.metrics
	r.mu.RUnlock()
	if mc == nil {
		return
	}
	mc.IncInvokeTotal(string(capability), operation, "ok")
	mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
}

func (r *Registry) emitEvents(ctx context.Context, capability hub.CapabilityType, operation string, result *InvokeResult) {
	r.mu.RLock()
	emitter := r.emitter
	r.mu.RUnlock()
	if emitter == nil || len(result.Events) == 0 {
		return
	}
	capt := string(capability)
	op := operation
	res := result
	submitEvent(capt, op, func() {
		emitter(context.WithoutCancel(ctx), res)
	})
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

	isMut := IsMutation(operation)
	skipCache := isMut || hasCursorParam(params)

	if !skipCache {
		if cached := cacheRead(capability, operation, params); cached != nil {
			return cached, nil
		}
	}

	ctx, span := trace.StartSpan(ctx, "ability."+string(capability)+"."+operation,
		attribute.String("capability.name", string(capability)),
		attribute.String("capability.operation", operation),
	)
	defer span.End()

	start := time.Now()
	var result *InvokeResult
	var err error
	invokeErr := bulkhead.Get(string(capability)).Do(ctx, func() error {
		var closureErr error
		result, closureErr = invoker(ctx, params)
		return closureErr
	})
	if invokeErr != nil {
		if errors.Is(invokeErr, bulkhead.ErrBulkheadFull) {
			err = types.Errorf(types.ErrRateLimited, "bulkhead full for %s: %v", capability, invokeErr)
		} else if errors.Is(invokeErr, bulkhead.ErrBulkheadTimeout) {
			err = types.Errorf(types.ErrTimeout, "bulkhead timeout for %s: %v", capability, invokeErr)
		} else {
			err = invokeErr
		}
		trace.RecordError(ctx, err)
		r.recordErrorMetrics(capability, operation, start, err)
		return nil, err
	}
	if result == nil {
		result = &InvokeResult{}
	}
	result.Capability = capability
	result.Operation = operation

	r.recordSuccessMetrics(capability, operation, start)
	r.emitEvents(ctx, capability, operation, result)

	if !skipCache {
		cacheWrite(capability, operation, params, result)
	}
	if isMut {
		cacheInvalidate(capability)
	}

	return result, nil
}
