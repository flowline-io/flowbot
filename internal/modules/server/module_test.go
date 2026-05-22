package server

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "name equals server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, "server", Name)
		})
	}
}

func TestBotInit(t *testing.T) {
	tests := []struct {
		name      string
		config    configType
		preInit   bool
		wantErr   bool
		wantReady bool
	}{
		{
			name:      "enabled config makes handler ready",
			config:    configType{Enabled: true},
			wantReady: true,
		},
		{
			name:      "disabled config makes handler not ready",
			config:    configType{Enabled: false},
			wantReady: false,
		},
		{
			name:    "invalid JSON returns error",
			wantErr: true,
		},
		{
			name:    "already initialized returns error",
			preInit: true,
			config:  configType{Enabled: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preInit {
				handler = moduleHandler{initialized: true}
			} else {
				handler = moduleHandler{}
			}

			var data json.RawMessage
			if tt.name == "invalid JSON returns error" {
				data = json.RawMessage(`{invalid`)
			} else if !tt.preInit || tt.config.Enabled {
				data, _ = sonic.Marshal(tt.config)
			} else {
				data, _ = sonic.Marshal(configType{Enabled: true})
			}

			err := handler.Init(data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantReady, handler.IsReady())
			}
		})
	}
}

func TestCommandRules(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "all expected commands are defined",
			fn: func(t *testing.T) {
				assert.NotEmpty(t, commandRules)

				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}

				assert.Contains(t, defines, "version")
				assert.Contains(t, defines, "mem stats")
				assert.Contains(t, defines, "golang stats")
				assert.Contains(t, defines, "server stats")
				assert.Contains(t, defines, "online stats")
				assert.Contains(t, defines, "adguard status")
				assert.Contains(t, defines, "adguard stats")
				assert.Contains(t, defines, "queue stats")
				assert.Contains(t, defines, "check")
			},
		},
		{
			name: "all command rules have handlers",
			fn: func(t *testing.T) {
				for _, r := range commandRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func TestWebserviceRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "webservice rules defined and at least two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, webserviceRules)
			assert.GreaterOrEqual(t, len(webserviceRules), 2)
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "handler returns two rulesets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 2)
		})
	}
}
