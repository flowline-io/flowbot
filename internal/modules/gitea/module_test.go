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

func TestWebhookRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should contain issue and repo webhooks with Secret=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, webhookRules)

			ids := make(map[string]bool)
			for _, r := range webhookRules {
				ids[r.Id] = true
			}

			assert.True(t, ids[IssueWebhookID])
			assert.True(t, ids[RepoWebhookID])

			for _, r := range webhookRules {
				assert.True(t, r.Secret, "webhook %q should have Secret=true", r.Id)
			}
		})
	}
}

func TestWebhookRules_HaveHandlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all webhook rules should have non-nil handlers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range webhookRules {
				assert.NotNil(t, r.Handler, "handler for webhook %q should not be nil", r.Id)
			}
		})
	}
}

func TestCronRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should contain gitea_metrics cron"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, cronRules)

			names := make(map[string]bool)
			for _, r := range cronRules {
				names[r.Name] = true
			}

			assert.True(t, names["gitea_metrics"])
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should return 2 rulesets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 2) // commandRules, webhookRules
		})
	}
}
