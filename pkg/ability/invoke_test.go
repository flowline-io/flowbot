package ability

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	assert.NotNil(t, r.handlers)
	assert.Empty(t, r.handlers)
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	invoker := func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{Data: "ok"}, nil
	}

	err := r.Register(hub.CapBookmark, "list", invoker)
	require.NoError(t, err)

	r.mu.RLock()
	require.Contains(t, r.handlers, hub.CapBookmark)
	require.Contains(t, r.handlers[hub.CapBookmark], "list")
	r.mu.RUnlock()
}

func TestRegistry_RegisterEmptyCapability(t *testing.T) {
	r := NewRegistry()

	err := r.Register("", "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrInvalidArgument))
	assert.Contains(t, err.Error(), "capability is required")
}

func TestRegistry_RegisterEmptyOperation(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrInvalidArgument))
	assert.Contains(t, err.Error(), "operation is required")
}

func TestRegistry_RegisterNilInvoker(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "list", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrInvalidArgument))
	assert.Contains(t, err.Error(), "invoker is required")
}

func TestRegistry_RegisterMultipleOperations(t *testing.T) {
	r := NewRegistry()

	invoker := func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{}, nil
	}

	require.NoError(t, r.Register(hub.CapBookmark, "list", invoker))
	require.NoError(t, r.Register(hub.CapBookmark, "get", invoker))
	require.NoError(t, r.Register(hub.CapBookmark, "create", invoker))

	r.mu.RLock()
	assert.Len(t, r.handlers[hub.CapBookmark], 3)
	r.mu.RUnlock()
}

func TestRegistry_InvokeSuccess(t *testing.T) {
	r := NewRegistry()

	expected := &InvokeResult{
		Data: "hello",
		Text: "some text",
		Meta: map[string]any{"key": "value"},
	}

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
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
}

func TestRegistry_InvokeCapabilityNotFound(t *testing.T) {
	r := NewRegistry()

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, types.ErrNotFound))
	assert.Contains(t, err.Error(), "capability bookmark not found")
}

func TestRegistry_InvokeOperationNotFound(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{}, nil
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "get", nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, types.ErrNotImplemented))
	assert.Contains(t, err.Error(), "operation bookmark.get not implemented")
}

func TestRegistry_InvokeNilParams(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		require.NotNil(t, params)
		assert.Empty(t, params)
		return &InvokeResult{Text: "ok"}, nil
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ok", result.Text)
}

func TestRegistry_InvokeNilResultBecomesEmpty(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return nil, nil
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, hub.CapBookmark, result.Capability)
	assert.Equal(t, "list", result.Operation)
}

func TestRegistry_InvokePropagatesError(t *testing.T) {
	r := NewRegistry()

	invokeErr := errors.New("something went wrong")
	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return nil, invokeErr
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, invokeErr, err)
}

func TestRegistry_InvokeEmitsEvents(t *testing.T) {
	r := NewRegistry()

	var emitted *InvokeResult
	var mu sync.Mutex
	r.emitter = func(ctx context.Context, result *InvokeResult) {
		mu.Lock()
		defer mu.Unlock()
		emitted = result
	}

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{
			Events: []EventRef{{EventID: "evt1", EventType: "bookmark.list", EntityID: "123"}},
		}, nil
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Wait for async goroutine
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, emitted)
	require.Len(t, emitted.Events, 1)
	assert.Equal(t, "evt1", emitted.Events[0].EventID)
}

func TestRegistry_InvokeNoEmitWithoutEvents(t *testing.T) {
	r := NewRegistry()

	emitted := false
	r.emitter = func(ctx context.Context, result *InvokeResult) {
		emitted = true
	}

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{}, nil
	})
	require.NoError(t, err)

	_, err = 	r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)
	assert.False(t, emitted)
}

func TestRegistry_InvokeNoEmitWithoutEmitter(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapBookmark, "list", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{
			Events: []EventRef{{EventID: "evt1", EventType: "test"}},
		}, nil
	})
	require.NoError(t, err)

	result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestSetEventEmitter(t *testing.T) {
	called := false
	SetEventEmitter(func(ctx context.Context, result *InvokeResult) {
		called = true
	})

	require.NotNil(t, DefaultRegistry.emitter)
	DefaultRegistry.emitter(t.Context(), &InvokeResult{})
	assert.True(t, called)

	DefaultRegistry.emitter = nil
}

func TestRegisterInvoker(t *testing.T) {
	invoker := func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{Data: "via convenience"}, nil
	}

	err := RegisterInvoker(hub.CapBookmark, "test_op", invoker)
	require.NoError(t, err)

	result, err := 	Invoke(t.Context(), hub.CapBookmark, "test_op", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "via convenience", result.Data)

	// Clean up default registry
	DefaultRegistry.handlers = make(map[hub.CapabilityType]map[string]Invoker)
}

func TestRegistry_InvokeResultHasCapabilityAndOperation(t *testing.T) {
	r := NewRegistry()

	err := r.Register(hub.CapArchive, "add", func(ctx context.Context, params map[string]any) (*InvokeResult, error) {
		return &InvokeResult{Text: "archived"}, nil
	})
	require.NoError(t, err)

	result, err := 	r.Invoke(t.Context(), hub.CapArchive, "add", nil)
	require.NoError(t, err)
	assert.Equal(t, hub.CapArchive, result.Capability)
	assert.Equal(t, "add", result.Operation)
	assert.Equal(t, "archived", result.Text)
}

func TestInvokeResult_EmptyDefaults(t *testing.T) {
	r := InvokeResult{}
	assert.Empty(t, r.Capability)
	assert.Empty(t, r.Operation)
	assert.Nil(t, r.Data)
	assert.Nil(t, r.Page)
	assert.Empty(t, r.Text)
	assert.Nil(t, r.Meta)
	assert.Nil(t, r.Events)
}
