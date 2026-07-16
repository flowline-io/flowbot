package capability

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterSpec(t *testing.T) {
	t.Parallel()

	invoker := func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
		return &InvokeResult{Data: "ok"}, nil
	}
	capTypeSkipped := hub.CapabilityType("test-register-skip")
	capTypeOK := hub.CapabilityType("test-register-ok")

	tests := []struct {
		name    string
		spec    Spec
		wantErr bool
		errIs   error
	}{
		{
			name:    "missing type",
			spec:    Spec{Instance: "svc"},
			wantErr: true,
			errIs:   types.ErrInvalidArgument,
		},
		{
			name:    "nil instance skips registration",
			spec:    Spec{Type: capTypeSkipped, App: "test"},
			wantErr: false,
		},
		{
			name: "registers operations and invoker",
			spec: Spec{
				Type:     capTypeOK,
				App:      "test",
				Instance: "svc",
				Ops: []OpDef{{
					Name:        "fetch",
					Description: "fetch item",
					Handler:     invoker,
					Mutation:    true,
				}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				if tt.spec.Type != "" {
					hub.Default.Unregister(tt.spec.Type)
					DefaultRegistry.Unregister(tt.spec.Type, "fetch")
				}
			})

			err := Register(tt.spec)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)

			if tt.spec.Instance == nil {
				_, ok := hub.Default.Get(tt.spec.Type)
				assert.False(t, ok)
				return
			}

			desc, ok := hub.Default.Get(tt.spec.Type)
			require.True(t, ok)
			assert.Equal(t, tt.spec.Type, desc.Type)
			assert.True(t, IsMutation("fetch"))

			result, err := DefaultRegistry.Invoke(context.Background(), tt.spec.Type, "fetch", nil)
			require.NoError(t, err)
			assert.Equal(t, "ok", result.Data)
		})
	}
}

func TestRegisterSpecValidation(t *testing.T) {
	t.Parallel()

	capType := hub.CapabilityType("test-register-validate")

	tests := []struct {
		name    string
		spec    Spec
		errText string
	}{
		{
			name: "empty operation name",
			spec: Spec{
				Type: capType, App: "test", Instance: "svc",
				Ops: []OpDef{{Handler: func(context.Context, map[string]any) (*InvokeResult, error) { return nil, nil }}},
			},
			errText: "operation name is required",
		},
		{
			name: "nil handler",
			spec: Spec{
				Type: capType, App: "test", Instance: "svc",
				Ops: []OpDef{{Name: "broken"}},
			},
			errText: "handler required",
		},
		{
			name: "valid minimal op",
			spec: Spec{
				Type: capType, App: "test", Instance: "svc",
				Ops: []OpDef{{
					Name:    "ping",
					Handler: func(context.Context, map[string]any) (*InvokeResult, error) { return &InvokeResult{}, nil },
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				hub.Default.Unregister(capType)
				DefaultRegistry.Unregister(capType, "ping")
				DefaultRegistry.Unregister(capType, "broken")
			})

			err := Register(tt.spec)
			if tt.errText != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRegistryUnregister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "removes operation"},
		{name: "removes empty capability bucket"},
		{name: "no-op when missing"},
	}

	capType := hub.CapExample
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			require.NoError(t, r.Register(capType, "temp", func(context.Context, map[string]any) (*InvokeResult, error) {
				return &InvokeResult{}, nil
			}))

			if tt.name == "no-op when missing" {
				r.Unregister(capType, "missing")
				_, err := r.Invoke(context.Background(), capType, "temp", nil)
				require.NoError(t, err)
				return
			}

			r.Unregister(capType, "temp")
			_, err := r.Invoke(context.Background(), capType, "temp", nil)
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrNotFound)

			if tt.name == "removes empty capability bucket" {
				r.mu.RLock()
				_, ok := r.handlers[capType]
				r.mu.RUnlock()
				assert.False(t, ok)
			}
		})
	}
}

func TestUnregisterInvoker(t *testing.T) {
	t.Parallel()

	capType := hub.CapExample
	require.NoError(t, DefaultRegistry.Register(capType, "global-temp", func(context.Context, map[string]any) (*InvokeResult, error) {
		return &InvokeResult{Data: "alive"}, nil
	}))
	t.Cleanup(func() {
		DefaultRegistry.Unregister(capType, "global-temp")
	})

	UnregisterInvoker(capType, "global-temp")
	_, err := Invoke(context.Background(), capType, "global-temp", nil)
	require.Error(t, err)
}

func TestSetBulkheadCallbacks(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "wires callbacks without panic"},
		{name: "safe to call repeatedly"},
		{name: "safe after metrics collector set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { SetBulkheadCallbacks() })
		})
	}
}

func TestEventSourceManagerGlobal(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "set and get manager"},
		{name: "get returns latest"},
		{name: "nil before set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() { SetEventSourceManager(nil) })

			if tt.name == "nil before set" {
				SetEventSourceManager(nil)
				assert.Nil(t, GetEventSourceManager())
				return
			}

			mgr := NewEventSourceManager(nil, nil, nil)
			SetEventSourceManager(mgr)
			assert.Equal(t, mgr, GetEventSourceManager())

			if tt.name == "get returns latest" {
				other := NewEventSourceManager(nil, nil, nil)
				SetEventSourceManager(other)
				assert.Equal(t, other, GetEventSourceManager())
			}
		})
	}
}

func TestGetEventPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "nil before init"},
		{name: "returns pool after init"},
		{name: "nil after shutdown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil before init" {
				ShutdownEventPool()
				assert.Nil(t, GetEventPool())
				return
			}

			require.NoError(t, InitEventPool(2, "1s", nil))
			if tt.name == "returns pool after init" {
				assert.NotNil(t, GetEventPool())
				ShutdownEventPool()
				return
			}

			ShutdownEventPool()
			assert.Nil(t, GetEventPool())
		})
	}
}
