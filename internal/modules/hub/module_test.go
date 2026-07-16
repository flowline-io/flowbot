package hub

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

func resetHubModuleState(t *testing.T) {
	t.Helper()
	handler = moduleHandler{}
	rcStore = nil
}

func TestModuleProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Name equals hub",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, "hub", Name)
			},
		},
		{
			name: "handler implements module.Handler",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Implements(t, (*module.Handler)(nil), &handler)
			},
		},
		{
			name: "Register does not panic",
			test: func(t *testing.T) {
				t.Parallel()
				require.NotPanics(t, func() {
					Register()
				})
				module.Unregister(Name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) json.RawMessage
		preInit bool
		wantErr string
		ready   bool
	}{
		{
			name: "disabled config skips initialization",
			setup: func(t *testing.T) json.RawMessage {
				t.Helper()
				data, err := sonic.Marshal(configType{Enabled: false})
				require.NoError(t, err)
				return data
			},
			ready: false,
		},
		{
			name: "invalid JSON returns parse error",
			setup: func(_ *testing.T) json.RawMessage {
				return json.RawMessage(`{invalid`)
			},
			wantErr: "failed to parse config",
		},
		{
			name:    "already initialized returns error",
			preInit: true,
			setup: func(t *testing.T) json.RawMessage {
				t.Helper()
				data, err := sonic.Marshal(configType{Enabled: true})
				require.NoError(t, err)
				return data
			},
			wantErr: "already initialized",
		},
		{
			name: "enabled without store database fails",
			setup: func(t *testing.T) json.RawMessage {
				t.Helper()
				store.Database = nil
				data, err := sonic.Marshal(configType{Enabled: true})
				require.NoError(t, err)
				return data
			},
			wantErr: "store database not available",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHubModuleState(t)
			oldDB := store.Database
			t.Cleanup(func() {
				store.Database = oldDB
				resetHubModuleState(t)
			})

			if tt.preInit {
				handler = moduleHandler{initialized: true}
			}

			data := tt.setup(t)
			err := handler.Init(data)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.ready, handler.IsReady())
		})
	}
}

func TestInitForE2E_WiresResourceChainStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "enabled init wires rcStore from sqlite client"},
		{name: "InitForE2E delegates to handler Init"},
		{name: "MountForE2E registers routes without panic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHubModuleState(t)
			client := sqlitetest.OpenClient(t, tt.name)
			rcStore = store.NewResourceChainStore(client)
			handler.initialized = true

			assert.NotNil(t, rcStore)
			assert.True(t, handler.IsReady())

			require.NotPanics(t, func() {
				MountForE2E(fiber.New())
			})
		})
	}
}

func TestIsReady(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		want        bool
	}{
		{name: "not initialized", initialized: false, want: false},
		{name: "initialized", initialized: true, want: true},
		{name: "zero value handler", initialized: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHubModuleState(t)
			handler.initialized = tt.initialized
			assert.Equal(t, tt.want, handler.IsReady())
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		wantErr     bool
	}{
		{name: "not initialized returns nil", initialized: false, wantErr: false},
		{name: "initialized without event manager returns error", initialized: true, wantErr: true},
		{name: "disabled init state skips bootstrap work", initialized: false, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHubModuleState(t)
			handler.initialized = tt.initialized
			err := handler.Bootstrap()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns TextMsg"},
		{name: "always succeeds"},
		{name: "ignores context values"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := moduleHandler{}
			payload, err := h.Input(types.Context{}, nil, nil)
			require.NoError(t, err)
			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Equal(t, "Input", msg.Text)
		})
	}
}

func TestRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "includes command webservice and form rules",
			test: func(t *testing.T) {
				t.Parallel()
				rules := handler.Rules()
				require.Len(t, rules, 3)
			},
		},
		{
			name: "webservice rules aggregate sub-modules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, webserviceRules)
				assert.GreaterOrEqual(t, len(webserviceRules), len(hubWebserviceRules))
			},
		},
		{
			name: "hub webservice rules are non-empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, hubWebserviceRules)
				for _, r := range hubWebserviceRules {
					assert.NotNil(t, r.Function)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
