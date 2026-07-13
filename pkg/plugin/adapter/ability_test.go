package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func TestAbilityAdapterRegister(t *testing.T) {
	tests := []struct {
		name    string
		ops     []string
		callErr string
		wantErr string
	}{
		{
			name: "registers list operation",
			ops:  []string{"list"},
		},
		{
			name: "registers multiple operations",
			ops:  []string{"list", "get", "create"},
		},
		{
			name: "no operations is valid",
			ops:  []string{},
		},
		{
			name:    "register fails on invoker error",
			ops:     []string{"fail_op"},
			callErr: "some error",
			wantErr: "", // Register doesn't fail at registration time, only at invocation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{
				callFn: func(_ context.Context, fn string, _ json.RawMessage) (json.RawMessage, error) {
					assert.Equal(t, "ability_call", fn)
					return json.RawMessage(`{"data": "result"}`), nil
				},
			}
			adapter := NewAbilityAdapter(runner, "test_adapter_reg", tt.ops)
			err := adapter.Register()
			defer adapter.Unregister()
			require.NoError(t, err)
		})
	}
}

func TestAbilityAdapterMakeInvoker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		op         string
		params     map[string]any
		resultJSON string
		wantData   any
		callErr    string
		wantErr    string
	}{
		{
			name:       "happy path get operation",
			op:         "get",
			params:     map[string]any{"id": "1"},
			resultJSON: `{"data": {"id": "1", "name": "item"}, "text": "found"}`,
			wantData:   map[string]any{"id": "1", "name": "item"},
		},
		{
			name:       "list operation returns data array",
			op:         "list",
			params:     map[string]any{"page_size": 10.0},
			resultJSON: `{"data": [{"id": "1"}, {"id": "2"}]}`,
			wantData:   []any{map[string]any{"id": "1"}, map[string]any{"id": "2"}},
		},
		{
			name:       "empty result",
			op:         "stats",
			params:     nil,
			resultJSON: `{}`,
			wantData:   nil,
		},
		{
			name:    "runner call error",
			op:      "get",
			params:  map[string]any{"id": "1"},
			callErr: "remote error",
			wantErr: "remote error",
		},
		{
			name:       "invalid json from runner",
			op:         "get",
			params:     map[string]any{},
			resultJSON: `not-json`,
			wantErr:    "unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{
				callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
					if tt.callErr != "" {
						return nil, fmt.Errorf("%s", tt.callErr)
					}
					return json.RawMessage(tt.resultJSON), nil
				},
			}

			adapter := NewAbilityAdapter(runner, "example", []string{tt.op})
			invoker := adapter.makeInvoker(tt.op)

			result, err := invoker(context.Background(), tt.params)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantData, result.Data)
		})
	}
}

func TestAbilityAdapterUnregister(t *testing.T) {
	runner := &stubRunner{
		callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		},
	}

	ops := []string{"op1", "op2", "op3"}
	adapter := NewAbilityAdapter(runner, "test_unreg", ops)
	err := adapter.Register()
	require.NoError(t, err)

	// Verify invokers are registered by invoking one
	result, err := capability.Invoke(context.Background(), "test_unreg", "op1", map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, result)

	adapter.Unregister()

	// Verify invokers are unregistered
	_, err = capability.Invoke(context.Background(), "test_unreg", "op1", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
