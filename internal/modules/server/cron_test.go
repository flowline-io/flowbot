package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

func TestCronRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "six cron rules",
			fn: func(t *testing.T) {
				assert.Len(t, cronRules, 6)
			},
		},
		{
			name: "all names are correct",
			fn: func(t *testing.T) {
				names := make(map[string]bool)
				for _, r := range cronRules {
					names[r.Name] = true
				}

				assert.True(t, names["server_user_online_change"])
				assert.True(t, names["docker_images_prune"])
				assert.True(t, names["docker_metrics"])
				assert.True(t, names["monitor_metrics"])
				assert.True(t, names["rules_updater"])
				assert.True(t, names["online_agent_checker"])
			},
		},
		{
			name: "all scopes are correct",
			fn: func(t *testing.T) {
				scopeMap := make(map[string]cron.CronScope)
				for _, r := range cronRules {
					scopeMap[r.Name] = r.Scope
				}

				assert.Equal(t, cron.CronScopeUser, scopeMap["server_user_online_change"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["docker_images_prune"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["docker_metrics"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["monitor_metrics"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["rules_updater"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["online_agent_checker"])
			},
		},
		{
			name: "all actions are not nil",
			fn: func(t *testing.T) {
				for _, r := range cronRules {
					assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
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
