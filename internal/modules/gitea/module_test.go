package gitea

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{name: "should equal gitea", expected: "gitea"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Name)
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		config  configType
		rawJSON json.RawMessage
		preInit bool
		wantErr bool
		ready   bool
	}{
		{
			name:    "enabled config",
			config:  configType{Enabled: true},
			wantErr: false,
			ready:   true,
		},
		{
			name:    "disabled config",
			config:  configType{Enabled: false},
			wantErr: false,
			ready:   false,
		},
		{
			name:    "invalid JSON",
			rawJSON: json.RawMessage(`{invalid`),
			wantErr: true,
			ready:   false,
		},
		{
			name:    "already initialized",
			rawJSON: json.RawMessage(`{"enabled":true}`),
			preInit: true,
			wantErr: true,
			ready:   false,
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
			if tt.rawJSON != nil {
				data = tt.rawJSON
			} else {
				d, _ := sonic.Marshal(tt.config)
				data = d
			}

			err := handler.Init(data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.ready, handler.IsReady())
			}
		})
	}
}

func TestCommandRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should contain gitea command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, commandRules)

			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "gitea")
		})
	}
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all command rules should have non-nil handlers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should return 1 ruleset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 1) // commandRules
		})
	}
}
