package notify

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "name equals notify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "notify", Name)
		})
	}
}

func TestBotInit(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "notify list, delete, and config commands defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, commandRules)

			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "notify list")
			assert.Contains(t, defines, "notify delete [string]")
			assert.Contains(t, defines, "notify config")
		})
	}
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestFormRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "create_notify form rule is defined with fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, formRules)

			found := false
			for _, r := range formRules {
				if r.Id == createNotifyFormID {
					found = true
					assert.True(t, r.IsLongTerm)
					assert.NotEmpty(t, r.Title)
					assert.NotEmpty(t, r.Field)
					assert.NotNil(t, r.Handler)
				}
			}
			assert.True(t, found, "create_notify form rule should be defined")
		})
	}
}

func TestCronRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "cron rules are defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, cronRules)
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
