package bookmark

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
		name     string
		expected string
	}{
		{name: "should equal bookmark", expected: "bookmark"},
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
		{name: "should not be empty and contain bookmark list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, commandRules)

			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "bookmark list")
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

func TestCronRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should contain all expected cron rules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, cronRules)

			names := make(map[string]bool)
			for _, r := range cronRules {
				names[r.Name] = true
			}

			assert.True(t, names["bookmarks_tag"])
			assert.True(t, names["bookmarks_metrics"])
			assert.True(t, names["bookmarks_search"])
			assert.True(t, names["bookmarks_task"])
			assert.True(t, names["bookmarks_tag_merge"])
		})
	}
}

func TestCronRules_HaveActions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all cron rules should have non-nil action and non-empty when"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range cronRules {
				assert.NotNil(t, r.Action, "action for cron %q should not be nil", r.Name)
				assert.NotEmpty(t, r.When, "when for cron %q should not be empty", r.Name)
			}
		})
	}
}

func TestEventRules_Defined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should contain all expected event IDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, eventRules)

			ids := make(map[string]bool)
			for _, r := range eventRules {
				ids[r.Id] = true
			}

			assert.True(t, ids[types.BookmarkArchiveBotEventID])
			assert.True(t, ids[types.BookmarkCreateBotEventID])
			assert.True(t, ids[types.ArchiveBoxAddBotEventID])
		})
	}
}

func TestEventRules_HaveHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all event rules should have non-nil handlers"},
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

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should return 4 rulesets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 4) // commandRules, cronRules, eventRules, webserviceRules
		})
	}
}
