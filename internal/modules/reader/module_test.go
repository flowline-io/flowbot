package reader

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
		{name: "name equals reader"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, "reader", Name)
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

func TestCommandRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "reader command defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, commandRules)

			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "reader")
		})
	}
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestCronRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "reader_metrics and reader_daily_summary defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, cronRules)

			names := make(map[string]bool)
			for _, r := range cronRules {
				names[r.Name] = true
			}

			assert.True(t, names["reader_metrics"])
			assert.True(t, names["reader_daily_summary"])
		})
	}
}

func TestCronRules_HaveActions(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all cron rules have action and when expression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range cronRules {
				assert.NotNil(t, r.Action, "action for cron %q should not be nil", r.Name)
				assert.NotEmpty(t, r.When, "when for cron %q should not be empty", r.Name)
			}
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "handler returns three rulesets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 3)
		})
	}
}
