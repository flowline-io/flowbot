package modules_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/plugin/adapter"
	"github.com/flowline-io/flowbot/pkg/plugin/manager"
	"github.com/flowline-io/flowbot/pkg/types"
)

// stubRunner implements plugin.Runner for tests.
type stubRunner struct {
	callResult json.RawMessage
	callError  error
}

func (*stubRunner) Load(_ context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	return &plugin.PluginInfo{Name: m.Name, Version: m.Version}, nil
}
func (*stubRunner) Start(_ context.Context, _ json.RawMessage) error { return nil }
func (*stubRunner) Stop(_ context.Context) error                     { return nil }
func (s *stubRunner) Call(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
	if s.callError != nil {
		return nil, s.callError
	}
	return s.callResult, nil
}
func (*stubRunner) Health(_ context.Context) (*plugin.HealthStatus, error) {
	return &plugin.HealthStatus{Ready: true}, nil
}

func TestModuleAdapterCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     json.RawMessage
		callError  error
		wantErrMsg string
		wantText   string
	}{
		{
			name:     "responds to commands through the adapter with KVMsg",
			result:   json.RawMessage(`{"_type": "KVMsg", "text": "hello from plugin"}`),
			wantText: "hello from plugin",
		},
		{
			name:       "handles plugin errors gracefully",
			result:     json.RawMessage(`{}`),
			callError:  fmt.Errorf("plugin error"),
			wantErrMsg: "plugin error",
		},
		{
			name:     "handles TextMsg type payload",
			result:   json.RawMessage(`{"_type": "TextMsg", "text": "plain text response"}`),
			wantText: "plain text response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{callResult: tt.result, callError: tt.callError}
			m := &plugin.Manifest{Name: "test", Runtime: plugin.RuntimeGRPC}
			a := adapter.NewModuleAdapter(m, runner)

			payload, err := a.Command(types.Context{}, "hello")
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, payload)
			assert.Equal(t, tt.wantText, findTextField(payload))
		})
	}
}

func findTextField(payload types.MsgPayload) string {
	switch v := payload.(type) {
	case types.TextMsg:
		return v.Text
	case types.KVMsg:
		if s, ok := v["text"].(string); ok {
			return s
		}
	}
	return ""
}

func TestPluginManagerBasics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fn      func(t *testing.T, mgr *manager.PluginManager) error
		wantErr string
	}{
		{
			name: "lists empty when no plugins loaded",
			fn: func(t *testing.T, mgr *manager.PluginManager) error {
				assert.Empty(t, mgr.List())
				return nil
			},
		},
		{
			name: "rejects unload of unknown plugin",
			fn: func(_ *testing.T, mgr *manager.PluginManager) error {
				return mgr.UnloadPlugin(context.Background(), "nonexistent")
			},
			wantErr: "not found",
		},
		{
			name: "rejects reload of unknown plugin",
			fn: func(_ *testing.T, mgr *manager.PluginManager) error {
				return mgr.ReloadPlugin(context.Background(), "nonexistent", &plugin.Manifest{}, nil)
			},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewPluginManager(manager.DefaultPluginConfig(), zerolog.Nop())
			err := tt.fn(t, mgr)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
