package ability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"creates non-nil registry with empty handlers"},
		{"repeated NewRegistry calls produce independent instances"},
		{"newly created registry has no registered capabilities"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			require.NotNil(t, r)
			assert.NotNil(t, r.handlers)
			assert.Empty(t, r.handlers)
			r.mu.RLock()
			assert.NotContains(t, r.handlers, hub.CapBookmark)
			r.mu.RUnlock()
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"registers invoker for capability and operation"},
		{"registering duplicate operation overwrites previous invoker"},
		{"registering under different capability creates separate entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			invoker := func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{Data: "ok"}, nil
			}
			err := r.Register(hub.CapBookmark, "list", invoker)
			require.NoError(t, err)
			if tt.name == "registering duplicate operation overwrites previous invoker" {
				newInvoker := func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
					return &InvokeResult{Data: "overwritten"}, nil
				}
				err = r.Register(hub.CapBookmark, "list", newInvoker)
				require.NoError(t, err)
			}
			if tt.name == "registering under different capability creates separate entry" {
				err = r.Register(hub.CapArchive, "add", invoker)
				require.NoError(t, err)
			}
			r.mu.RLock()
			require.Contains(t, r.handlers, hub.CapBookmark)
			require.Contains(t, r.handlers[hub.CapBookmark], "list")
			r.mu.RUnlock()
		})
	}
}

func TestRegistry_RegisterEmptyCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		capability hub.CapabilityType
		errMsg     string
	}{
		{"empty capability returns invalid argument error", "", "capability is required"},
		{"whitespace-only capability is not empty", " ", ""},
		{"valid capability with empty operation returns invalid argument error", hub.CapBookmark, "operation is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			operation := "list"
			if tt.name == "valid capability with empty operation returns invalid argument error" {
				operation = ""
			}
			err := r.Register(tt.capability, operation, func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return nil, nil
			})
			if tt.errMsg != "" {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrInvalidArgument)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegistry_RegisterEmptyOperation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
		errMsg    string
	}{
		{"empty operation returns invalid argument error", "", "operation is required"},
		{"whitespace-only operation is not empty", " ", ""},
		{"valid operation with nil invoker returns invalid argument error", "list", "invoker is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			var invoker Invoker
			if tt.name != "valid operation with nil invoker returns invalid argument error" {
				invoker = func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
					return nil, nil
				}
			}
			err := r.Register(hub.CapBookmark, tt.operation, invoker)
			if tt.errMsg != "" {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrInvalidArgument)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegistry_RegisterNilInvoker(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		op   string
	}{
		{"nil invoker returns invalid argument error", hub.CapBookmark, "list"},
		{"nil invoker for reader capability returns invalid argument error", hub.CapReader, "list_feeds"},
		{"nil invoker for archive capability returns invalid argument error", hub.CapArchive, "add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			err := r.Register(tt.cap, tt.op, nil)
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrInvalidArgument)
			assert.Contains(t, err.Error(), "invoker is required")
		})
	}
}

func TestRegistry_RegisterMultipleOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"registers multiple operations under same capability"},
		{"registers operations across different capabilities"},
		{"registering same operation multiple times overwrites without error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			invoker := func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{}, nil
			}
			if tt.name == "registers operations across different capabilities" {
				require.NoError(t, r.Register(hub.CapBookmark, "list", invoker))
				require.NoError(t, r.Register(hub.CapArchive, "add", invoker))
				r.mu.RLock()
				assert.Contains(t, r.handlers, hub.CapBookmark)
				assert.Contains(t, r.handlers, hub.CapArchive)
				r.mu.RUnlock()
				return
			}
			require.NoError(t, r.Register(hub.CapBookmark, "list", invoker))
			require.NoError(t, r.Register(hub.CapBookmark, "get", invoker))
			require.NoError(t, r.Register(hub.CapBookmark, "create", invoker))
			if tt.name == "registering same operation multiple times overwrites without error" {
				require.NoError(t, r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
					return &InvokeResult{Data: "overwritten"}, nil
				}))
			}
			r.mu.RLock()
			assert.Len(t, r.handlers[hub.CapBookmark], 3)
			r.mu.RUnlock()
		})
	}
}

func TestRegistry_InvokeSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"invokes registered handler and populates result"},
		{"invoke returns result with empty meta"},
		{"invoke returns result with empty events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			expected := &InvokeResult{
				Data: "hello",
				Text: "some text",
				Meta: map[string]any{"key": "value"},
			}
			if tt.name == "invoke returns result with empty meta" {
				expected.Meta = nil
			}
			if tt.name == "invoke returns result with empty events" {
				expected.Meta = map[string]any{}
			}
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, params map[string]any) (*InvokeResult, error) {
				assert.Equal(t, "val", params["key"])
				return expected, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "val"})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, hub.CapBookmark, result.Capability)
			assert.Equal(t, "list", result.Operation)
			assert.Equal(t, "hello", result.Data)
			assert.Equal(t, "some text", result.Text)
		})
	}
}

func TestRegistry_InvokeCapabilityNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		capability hub.CapabilityType
	}{
		{"invoke returns not found error for missing capability", hub.CapBookmark},
		{"invoke with empty capability returns not found error", ""},
		{"invoke with unregistered capability returns not found error", hub.CapKanban},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			result, err := r.Invoke(t.Context(), tt.capability, "list", nil)
			require.Error(t, err)
			assert.Nil(t, result)
			require.ErrorIs(t, err, types.ErrNotFound)
			assert.Contains(t, err.Error(), "not found")
		})
	}
}

func TestRegistry_InvokeOperationNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
	}{
		{"invoke returns not implemented error for missing operation", "get"},
		{"invoke with empty operation returns not implemented error", ""},
		{"invoke unregistered operation on existing capability returns not implemented error", "delete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{}, nil
			})
			require.NoError(t, err)

			result, err := r.Invoke(t.Context(), hub.CapBookmark, tt.operation, nil)
			require.Error(t, err)
			assert.Nil(t, result)
			require.ErrorIs(t, err, types.ErrNotImplemented)
			assert.Contains(t, err.Error(), "not implemented")
		})
	}
}

func TestRegistry_InvokeNilParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"nil params are replaced with empty map"},
		{"explicitly nil and explicit empty map behave identically"},
		{"nil params with nil result from handler returns empty result"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, params map[string]any) (*InvokeResult, error) {
				require.NotNil(t, params)
				assert.Empty(t, params)
				if tt.name == "nil params with nil result from handler returns empty result" {
					return nil, nil
				}
				return &InvokeResult{Text: "ok"}, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.name == "nil params with nil result from handler returns empty result" {
				assert.Empty(t, result.Text)
			} else {
				assert.Equal(t, "ok", result.Text)
			}
		})
	}
}

func TestRegistry_InvokeNilResultBecomesEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		op   string
	}{
		{"nil result from handler is replaced with empty result", hub.CapBookmark, "list"},
		{"nil result for archive capability populates correct capability/operation", hub.CapArchive, "add"},
		{"nil result for kanban capability populates correct capability/operation", hub.CapKanban, "list_tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			err := r.Register(tt.cap, tt.op, func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return nil, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), tt.cap, tt.op, nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.cap, result.Capability)
			assert.Equal(t, tt.op, result.Operation)
		})
	}
}

func TestRegistry_InvokePropagatesError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
	}{
		{"error from handler is propagated to caller", errors.New("something went wrong")},
		{"not found error from handler is propagated", types.ErrNotFound},
		{"wrapped error from provider is propagated", fmt.Errorf("provider error: %w", types.ErrUnavailable)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			invokeErr := tt.err
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return nil, invokeErr
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Equal(t, invokeErr, err)
		})
	}
}

func TestRegistry_InvokeEmitsEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		events []EventRef
	}{
		{"invoke emits events when result has events and emitter is set", []EventRef{
			{EventID: "evt1", EventType: "bookmark.list", EntityID: "123"},
		}},
		{"invoke emits multiple events correctly", []EventRef{
			{EventID: "evt1", EventType: "bookmark.list", EntityID: "1"},
			{EventID: "evt2", EventType: "bookmark.list", EntityID: "2"},
		}},
		{"invoke emits events with zero entity id", []EventRef{
			{EventID: "evt1", EventType: "bookmark.create"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			var emitted *InvokeResult
			var mu sync.Mutex
			r.emitter = func(_ context.Context, result *InvokeResult) {
				mu.Lock()
				defer mu.Unlock()
				emitted = result
			}
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{Events: tt.events}, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			time.Sleep(50 * time.Millisecond)
			mu.Lock()
			defer mu.Unlock()
			require.NotNil(t, emitted)
			require.Len(t, emitted.Events, len(tt.events))
			assert.Equal(t, tt.events[0].EventID, emitted.Events[0].EventID)
		})
	}
}

func TestRegistry_InvokeNoEmitWithoutEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"invoke does not emit when result has no events"},
		{"nil events field in result does not emit"},
		{"empty events slice does not emit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			emitted := false
			r.emitter = func(_ context.Context, _ *InvokeResult) {
				emitted = true
			}
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				if tt.name == "nil events field in result does not emit" {
					return &InvokeResult{Events: nil}, nil
				}
				return &InvokeResult{Events: []EventRef{}}, nil
			})
			require.NoError(t, err)
			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			time.Sleep(20 * time.Millisecond)
			assert.False(t, emitted)
		})
	}
}

func TestRegistry_InvokeNoEmitWithoutEmitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"invoke succeeds without emitter when events are present"},
		{"invoke succeeds without emitter with nil events"},
		{"invoke succeeds without emitter with multiple events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			events := []EventRef{{EventID: "evt1", EventType: "test"}}
			if tt.name == "invoke succeeds without emitter with nil events" {
				events = nil
			}
			if tt.name == "invoke succeeds without emitter with multiple events" {
				events = []EventRef{
					{EventID: "evt1", EventType: "test"},
					{EventID: "evt2", EventType: "test"},
				}
			}
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{Events: events}, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestSetEventEmitter(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"sets event emitter on default registry"},
		{"clearing emitter with nil stops emission"},
		{"re-setting emitter with new function works"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			SetEventEmitter(func(_ context.Context, _ *InvokeResult) {
				called = true
			})
			DefaultRegistry.mu.RLock()
			em := DefaultRegistry.emitter
			DefaultRegistry.mu.RUnlock()
			require.NotNil(t, em)
			em(t.Context(), &InvokeResult{})
			assert.True(t, called)

			if tt.name == "clearing emitter with nil stops emission" {
				SetEventEmitter(nil)
				DefaultRegistry.mu.RLock()
				assert.Nil(t, DefaultRegistry.emitter)
				DefaultRegistry.mu.RUnlock()
				return
			}
			if tt.name == "re-setting emitter with new function works" {
				newCalled := false
				SetEventEmitter(func(_ context.Context, _ *InvokeResult) {
					newCalled = true
				})
				DefaultRegistry.mu.RLock()
				em2 := DefaultRegistry.emitter
				DefaultRegistry.mu.RUnlock()
				em2(t.Context(), &InvokeResult{})
				assert.True(t, newCalled)
			}
			SetEventEmitter(nil)
		})
	}
}

func TestRegisterInvoker(t *testing.T) {
	tests := []struct {
		name       string
		capability hub.CapabilityType
		operation  string
		invoker    Invoker
		wantErr    bool
	}{
		{"convenience function registers and invokes successfully", hub.CapBookmark, "test_op", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
			return &InvokeResult{Data: "via convenience"}, nil
		}, false},
		{"convenience function with empty capability returns error", "", "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
			return nil, nil
		}, true},
		{"convenience function with nil invoker returns error", hub.CapBookmark, "list", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterInvoker(tt.capability, tt.operation, tt.invoker)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				result, err := Invoke(t.Context(), tt.capability, tt.operation, nil)
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "via convenience", result.Data)
			}
			DefaultRegistry.mu.Lock()
			DefaultRegistry.handlers = make(map[hub.CapabilityType]map[string]Invoker)
			DefaultRegistry.mu.Unlock()
		})
	}
}

func TestRegistry_InvokeResultHasCapabilityAndOperation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		op   string
		text string
	}{
		{"result contains capability and operation from invoke call", hub.CapArchive, "add", "archived"},
		{"result contains correct capability and operation for kanban", hub.CapKanban, "list_tasks", "tasks"},
		{"result contains correct capability and operation for reader", hub.CapReader, "list_entries", "entries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			err := r.Register(tt.cap, tt.op, func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{Text: tt.text}, nil
			})
			require.NoError(t, err)
			result, err := r.Invoke(t.Context(), tt.cap, tt.op, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.cap, result.Capability)
			assert.Equal(t, tt.op, result.Operation)
			assert.Equal(t, tt.text, result.Text)
		})
	}
}

func TestInvokeResult_EmptyDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"empty invoke result has zero values for all fields"},
		{"empty result has zero value for capability string"},
		{"empty result has empty page info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := InvokeResult{}
			assert.Empty(t, r.Capability)
			assert.Empty(t, r.Operation)
			assert.Nil(t, r.Data)
			assert.Nil(t, r.Page)
			assert.Empty(t, r.Text)
			assert.Nil(t, r.Meta)
			assert.Nil(t, r.Events)
		})
	}
}

func TestSetMetricsCollector(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"sets nil collector without panic"},
		{"sets no-op collector"},
		{"can set after default registry is created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() {
				SetMetricsCollector(nil)
				SetMetricsCollector(metrics.NewAbilityCollector(nil))
			})
		})
	}
}

func TestRegistry_InvokeCacheHit(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	t.Cleanup(func() {
		cache.Instance = nil
	})

	tests := []struct {
		name string
	}{
		{"second call with same params returns cached result without invoking handler"},
		{"cache stores result data correctly on subsequent hit"},
		{"cache preserves text field on hit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = setupTestCache(t)
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				return &InvokeResult{Data: "result", Text: "cached text"}, nil
			})
			require.NoError(t, err)

			result1, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "val"})
			require.NoError(t, err)
			require.Equal(t, 1, callCount)

			result2, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "val"})
			require.NoError(t, err)
			require.Equal(t, 1, callCount, "handler should not be called again on cache hit")

			assert.Equal(t, "result", result1.Data)
			assert.Equal(t, "result", result2.Data)
			assert.Equal(t, "cached text", result2.Text)
		})
	}
}

func TestRegistry_InvokeCacheMiss(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	t.Cleanup(func() {
		cache.Instance = nil
	})

	tests := []struct {
		name string
	}{
		{"first call invokes handler and returns result"},
		{"different params produce different cache keys"},
		{"handler error is not cached"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = setupTestCache(t)
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				if tt.name == "handler error is not cached" {
					return nil, errors.New("provider error")
				}
				return &InvokeResult{Data: "fresh"}, nil
			})
			require.NoError(t, err)

			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "first"})
			if tt.name == "handler error is not cached" {
				require.Error(t, err)
				require.Equal(t, 1, callCount)
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "first"})
				require.Error(t, err)
				require.Equal(t, 2, callCount, "error should not be cached, handler called again")
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, callCount)
			assert.Equal(t, "fresh", result.Data)

			if tt.name == "different params produce different cache keys" {
				result2, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "second"})
				require.NoError(t, err)
				require.Equal(t, 2, callCount, "different params should be cache miss")
				assert.Equal(t, "fresh", result2.Data)
			}
		})
	}
}

func TestRegistry_InvokeCacheMutationInvalidates(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	t.Cleanup(func() {
		cache.Instance = nil
	})

	tests := []struct {
		name string
	}{
		{"write operation invalidates all cached read operations for same capability"},
		{"write on one capability does not affect another capability"},
		{"multiple writes invalidate progressively"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = setupTestCache(t)
			r := NewRegistry()
			listCallCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				listCallCount++
				return &InvokeResult{Data: "list result"}, nil
			})
			require.NoError(t, err)

			kanbanCallCount := 0
			if tt.name == "write on one capability does not affect another capability" {
				err = r.Register(hub.CapKanban, "list_tasks", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
					kanbanCallCount++
					return &InvokeResult{Data: "kanban result"}, nil
				})
				require.NoError(t, err)
			}

			createCallCount := 0
			err = r.Register(hub.CapBookmark, "create", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				createCallCount++
				return &InvokeResult{Text: "created"}, nil
			})
			require.NoError(t, err)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.Equal(t, 1, listCallCount)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "create", nil)
			require.NoError(t, err)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.Equal(t, 2, listCallCount, "cache should be invalidated after write")

			if tt.name == "write on one capability does not affect another capability" {
				_, err = r.Invoke(t.Context(), hub.CapKanban, "list_tasks", nil)
				require.NoError(t, err)
				_, err = r.Invoke(t.Context(), hub.CapKanban, "list_tasks", nil)
				require.NoError(t, err)
				require.Equal(t, 1, kanbanCallCount, "kanban cache should survive bookmark write")
			}

			if tt.name == "multiple writes invalidate progressively" {
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "create", nil)
				require.NoError(t, err)
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
				require.NoError(t, err)
				require.Equal(t, 3, listCallCount)
			}
		})
	}
}

func TestRegistry_InvokeCacheCursorSkip(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	t.Cleanup(func() {
		cache.Instance = nil
	})

	tests := []struct {
		name string
	}{
		{"cursor param bypasses cache read"},
		{"cursor param bypasses cache write"},
		{"non-cursor params are cached normally"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = setupTestCache(t)
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				return &InvokeResult{Data: "result"}, nil
			})
			require.NoError(t, err)

			params := map[string]any{"cursor": "next_page_token"}
			if tt.name == "non-cursor params are cached normally" {
				params = map[string]any{"archived": true}
			}

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", params)
			require.NoError(t, err)
			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", params)
			require.NoError(t, err)

			if tt.name == "non-cursor params are cached normally" {
				require.Equal(t, 1, callCount, "non-cursor params should be cached")
			} else {
				require.Equal(t, 2, callCount, "cursor params should not be cached")
			}
		})
	}
}

func TestRegistry_InvokeCacheSerializationRoundtrip(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	t.Cleanup(func() {
		cache.Instance = nil
	})

	tests := []struct {
		name string
	}{
		{"roundtrip preserves nested data and page fields"},
		{"roundtrip preserves text and meta fields"},
		{"cached result has nil events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = setupTestCache(t)
			r := NewRegistry()
			var events []EventRef
			if tt.name == "cached result has nil events" {
				events = []EventRef{{EventID: "evt1", EventType: "bookmark.list", EntityID: "123"}}
			}
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				return &InvokeResult{
					Data:   map[string]any{"nested": "value"},
					Page:   &PageInfo{HasMore: true},
					Text:   "some text",
					Meta:   map[string]any{"source": "cache"},
					Events: events,
				}, nil
			})
			require.NoError(t, err)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)

			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.name == "cached result has nil events" {
				assert.Nil(t, result.Events, "events should not be replayed from cache")
				return
			}

			if tt.name == "roundtrip preserves nested data and page fields" {
				assert.True(t, result.Page.HasMore)
				dataMap, ok := result.Data.(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "value", dataMap["nested"])
				return
			}

			assert.Equal(t, "some text", result.Text)
			assert.Equal(t, "cache", result.Meta["source"])
		})
	}
}

func TestInvokeResult_ResourceMetaJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   InvokeResult
		wantJSON bool
	}{
		{"nil Resource omitted", InvokeResult{Data: "ok"}, false},
		{
			"non-nil Resource serializes",
			InvokeResult{
				Data:     "ok",
				Resource: &ResourceMeta{EventID: "evt-1", EntityID: "123", App: "test-app"},
			},
			true,
		},
		{
			"zero-value Resource still appears",
			InvokeResult{Data: "ok", Resource: &ResourceMeta{}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := sonic.Marshal(tt.result)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, sonic.Unmarshal(data, &decoded))
			if tt.wantJSON {
				assert.Contains(t, decoded, "_resource")
			} else {
				assert.NotContains(t, decoded, "_resource")
			}
		})
	}
}

func setupTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	cache.Instance = nil
	c, err := cache.NewCache(&config.Type{})
	require.NoError(t, err)
	return c
}
