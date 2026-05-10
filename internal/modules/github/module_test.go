package github

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
		name     string
		expected string
	}{
		{name: "should equal github", expected: "github"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, Name)
		})
	}
}

func TestInit(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should contain expected command defines"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, commandRules)

			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "github setting")
			assert.Contains(t, defines, "github oauth")
			assert.Contains(t, defines, "deploy")
		})
	}
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all command rules should have non-nil handlers"},
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

func TestWebhookRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should contain PackageWebhookID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, webhookRules)

			ids := make(map[string]bool)
			for _, r := range webhookRules {
				ids[r.Id] = true
			}

			assert.True(t, ids[PackageWebhookID])
		})
	}
}

func TestWebhookRules_HaveHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all webhook rules should have non-nil handlers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range webhookRules {
				assert.NotNil(t, r.Handler, "handler for webhook %q should not be nil", r.Id)
			}
		})
	}
}

func TestCronRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should contain expected cron names"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, cronRules)

			names := make(map[string]bool)
			for _, r := range cronRules {
				names[r.Name] = true
			}

			assert.True(t, names["github_starred"])
			assert.True(t, names["github_notifications"])
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should return 3 rulesets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 3) // commandRules, formRules, webhookRules
		})
	}
}
