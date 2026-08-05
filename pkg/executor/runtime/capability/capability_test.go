package capability

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func registerTempInvoker(t *testing.T, capType hub.CapabilityType, op string, fn capability.Invoker) {
	t.Helper()
	require.NoError(t, capability.RegisterInvoker(capType, op, fn))
	t.Cleanup(func() { capability.UnregisterInvoker(capType, op) })
}

func TestRuntime_Run(t *testing.T) {
	const capType = hub.CapabilityType("runtime_cap_test")
	const op = "echo"

	rt := New()
	tests := []struct {
		name       string
		task       *types.Task
		register   bool
		invokeErr  error
		invokeData map[string]any
		wantErr    string
		check      func(t *testing.T, task *types.Task, gotParams map[string]any)
	}{
		{
			name:    "invalid action without dot",
			task:    &types.Task{Run: "capability:nodot"},
			wantErr: "invalid capability action",
		},
		{
			name:     "invalid params json",
			task:     &types.Task{Run: Prefix + string(capType) + "." + op, Env: map[string]string{"CAPABILITY_PARAMS": "{"}},
			wantErr:  "decode capability params",
			register: true,
		},
		{
			name: "injects uid and topic",
			task: &types.Task{
				Run: Prefix + string(capType) + "." + op,
				Env: map[string]string{
					"CAPABILITY_PARAMS": `{"k":"v"}`,
					"CAPABILITY_UID":    "u1",
					"CAPABILITY_TOPIC":  "t1",
				},
			},
			register:   true,
			invokeData: map[string]any{"ok": true},
			check: func(t *testing.T, task *types.Task, gotParams map[string]any) {
				t.Helper()
				assert.Equal(t, "v", gotParams["k"])
				assert.Equal(t, "u1", gotParams["_uid"])
				assert.Equal(t, "t1", gotParams["_topic"])
				var out map[string]any
				require.NoError(t, sonic.Unmarshal([]byte(task.Result), &out))
				assert.Equal(t, true, out["data"].(map[string]any)["ok"])
			},
		},
		{
			name: "parses dotted capability type",
			task: &types.Task{
				Run: Prefix + "a.b.c." + op,
				Env: map[string]string{},
			},
			register: true,
			check: func(t *testing.T, _ *types.Task, _ map[string]any) {
				t.Helper()
				// registered under a.b.c below via special path — see subtest setup
			},
		},
		{
			name:      "wraps invoke error",
			task:      &types.Task{Run: Prefix + string(capType) + "." + op},
			register:  true,
			invokeErr: errors.New("boom"),
			wantErr:   string(capType) + "." + op + ":",
		},
		{
			name: "empty params env yields empty map",
			task: &types.Task{
				Run: Prefix + string(capType) + "." + op,
				Env: map[string]string{"CAPABILITY_PARAMS": ""},
			},
			register:   true,
			invokeData: map[string]any{},
			check: func(t *testing.T, task *types.Task, gotParams map[string]any) {
				t.Helper()
				assert.Empty(t, gotParams)
				assert.NotEmpty(t, task.Result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotParams map[string]any
			if tt.name == "parses dotted capability type" {
				registerTempInvoker(t, hub.CapabilityType("a.b.c"), op, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
					gotParams = params
					return &capability.InvokeResult{Data: map[string]any{"nested": true}}, nil
				})
			} else if tt.register {
				registerTempInvoker(t, capType, op, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
					gotParams = params
					if tt.invokeErr != nil {
						return nil, tt.invokeErr
					}
					return &capability.InvokeResult{Data: tt.invokeData}, nil
				})
			}

			err := rt.Run(context.Background(), tt.task)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.name == "parses dotted capability type" {
				assert.NotNil(t, gotParams)
				assert.Contains(t, tt.task.Result, "nested")
				return
			}
			if tt.check != nil {
				tt.check(t, tt.task, gotParams)
			}
		})
	}
}

func TestRuntime_Noops(t *testing.T) {
	rt := New()
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "stop", fn: func() error { return rt.Stop(context.Background(), &types.Task{ID: "x"}) }},
		{name: "health", fn: func() error { return rt.HealthCheck(context.Background()) }},
		{name: "close", fn: func() error { return rt.Close() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.fn())
		})
	}
}
