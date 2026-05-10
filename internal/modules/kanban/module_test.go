package kanban

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestBotName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "name equals kanban"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "kanban", Name)
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

func TestCommandRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "at least one command rule defined",
			fn: func(t *testing.T) {
				assert.NotEmpty(t, commandRules)
				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}
				assert.Contains(t, defines, "kanban status")
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
			t.Parallel()
			tt.fn(t)
		})
	}
}

func TestCronRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "kanban_metrics cron rule is defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, cronRules)

			names := make(map[string]bool)
			for _, r := range cronRules {
				names[r.Name] = true
			}

			assert.True(t, names["kanban_metrics"])
		})
	}
}

func TestEventRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "task create bot event rule is defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, eventRules)

			ids := make(map[string]bool)
			for _, r := range eventRules {
				ids[r.Id] = true
			}

			assert.True(t, ids[types.TaskCreateBotEventID])
		})
	}
}

func TestEventRules_HaveHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all event rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range eventRules {
				assert.NotNil(t, r.Handler, "handler for event %q should not be nil", r.Id)
			}
		})
	}
}

func TestWebhookRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "at least one webhook rule defined",
			fn: func(t *testing.T) {
				assert.NotEmpty(t, webhookRules)

				ids := make(map[string]bool)
				for _, r := range webhookRules {
					ids[r.Id] = true
				}

				assert.True(t, ids[KanbanWebhookID])
			},
		},
		{
			name: "all webhook rules have handlers",
			fn: func(t *testing.T) {
				for _, r := range webhookRules {
					assert.NotNil(t, r.Handler, "handler for webhook %q should not be nil", r.Id)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "handler returns five rulesets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 5)
		})
	}
}
