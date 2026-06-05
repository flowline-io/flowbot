package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/types"
)

// stubRunner implements plugin.Runner for testing.
type stubRunner struct {
	callFn func(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error)
}

func (*stubRunner) Load(_ context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	return &plugin.PluginInfo{Name: m.Name, Version: m.Version}, nil
}
func (*stubRunner) Start(_ context.Context, _ json.RawMessage) error { return nil }
func (*stubRunner) Stop(_ context.Context) error                     { return nil }
func (s *stubRunner) Call(_ context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	if s.callFn != nil {
		return s.callFn(context.Background(), function, params)
	}
	return json.RawMessage(`{}`), nil
}
func (*stubRunner) Health(_ context.Context) (*plugin.HealthStatus, error) {
	return &plugin.HealthStatus{Ready: true}, nil
}

func TestModuleAdapterCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  string
		callErr string
		wantErr string
		want    types.MsgPayload
	}{
		{
			name:   "happy path text msg",
			result: `{"_type": "TextMsg", "text": "hello from plugin"}`,
			want:   types.TextMsg{Text: "hello from plugin"},
		},
		{
			name:   "happy path kv msg",
			result: `{"_type": "KVMsg", "key": "value"}`,
			want:   types.KVMsg{"key": "value"},
		},
		{
			name:   "fallback to TextMsg without _type",
			result: `{"text": "plain text"}`,
			want:   types.TextMsg{Text: "plain text"},
		},
		{
			name:   "fallback to KVMsg without text",
			result: `{"some": "data"}`,
			want:   types.KVMsg{"some": "data"},
		},
		{
			name:    "plugin call error",
			result:  `{}`,
			callErr: "simulated error",
			wantErr: "simulated error",
		},
		{
			name:    "nil runner",
			wantErr: "no runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r plugin.Runner
			if tt.wantErr != "no runner" {
				r = &stubRunner{
					callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
						if tt.callErr != "" {
							return nil, fmt.Errorf("%s", tt.callErr)
						}
						return json.RawMessage(tt.result), nil
					},
				}
			}
			m := &plugin.Manifest{Name: "test"}
			adapter := NewModuleAdapter(m, r)

			payload, err := adapter.Command(types.Context{}, "hello")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, payload)
		})
	}
}

func TestModuleAdapterSwapRunner(t *testing.T) {
	t.Parallel()

	runner1 := &stubRunner{callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"_type": "TextMsg", "text": "runner1"}`), nil
	}}
	runner2 := &stubRunner{callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"_type": "TextMsg", "text": "runner2"}`), nil
	}}

	adapter := NewModuleAdapter(&plugin.Manifest{Name: "test"}, runner1)
	payload, _ := adapter.Command(types.Context{}, "hello")
	assert.Equal(t, types.TextMsg{Text: "runner1"}, payload)

	adapter.SwapRunner(runner2)
	payload, _ = adapter.Command(types.Context{}, "hello")
	assert.Equal(t, types.TextMsg{Text: "runner2"}, payload)
}

func TestModuleAdapterBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		callErr string
		wantErr string
	}{
		{
			name: "successful bootstrap",
		},
		{
			name:    "bootstrap fails",
			callErr: "bootstrap error",
			wantErr: "bootstrap error",
		},
		{
			name:    "nil runner",
			wantErr: "no runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r plugin.Runner
			if tt.wantErr != "no runner" {
				r = &stubRunner{
					callFn: func(_ context.Context, fn string, _ json.RawMessage) (json.RawMessage, error) {
						assert.Equal(t, "bootstrap", fn)
						if tt.callErr != "" {
							return nil, fmt.Errorf("%s", tt.callErr)
						}
						return json.RawMessage(`{}`), nil
					},
				}
			}
			adapter := NewModuleAdapter(&plugin.Manifest{Name: "test"}, r)
			err := adapter.Bootstrap()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestModuleAdapterInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		nilRun  bool
		wantErr string
	}{
		{
			name: "successful init",
		},
		{
			name:    "nil runner",
			nilRun:  true,
			wantErr: "no runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r plugin.Runner
			if !tt.nilRun {
				r = &stubRunner{}
			}

			adapter := NewModuleAdapter(&plugin.Manifest{Name: "test"}, r)
			err := adapter.Init(json.RawMessage(`{"enabled": true}`))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, adapter.IsReady())
		})
	}
}

func TestModuleAdapterIsReady(t *testing.T) {
	t.Parallel()

	adapter := NewModuleAdapter(&plugin.Manifest{Name: "test"}, &stubRunner{})
	assert.False(t, adapter.IsReady())

	_ = adapter.Init(json.RawMessage(`{}`))
	assert.True(t, adapter.IsReady())
}
