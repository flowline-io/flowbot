package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

func TestCronRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 cron rule",
			test: func(t *testing.T) {
				assert.Len(t, cronRules, 1)
			},
		},
		{
			name: "should contain hub_health_check",
			test: func(t *testing.T) {
				names := make(map[string]bool)
				for _, r := range cronRules {
					names[r.Name] = true
				}

				assert.True(t, names["hub_health_check"])
			},
		},
		{
			name: "should have system scope",
			test: func(t *testing.T) {
				scopeMap := make(map[string]cron.CronScope)
				for _, r := range cronRules {
					scopeMap[r.Name] = r.Scope
				}

				assert.Equal(t, cron.CronScopeSystem, scopeMap["hub_health_check"])
			},
		},
		{
			name: "should have correct cron expression",
			test: func(t *testing.T) {
				whenMap := make(map[string]string)
				for _, r := range cronRules {
					whenMap[r.Name] = r.When
				}

				assert.Equal(t, "*/5 * * * *", whenMap["hub_health_check"])
			},
		},
		{
			name: "all crons should have non-nil actions",
			test: func(t *testing.T) {
				for _, r := range cronRules {
					assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
				}
			},
		},
		{
			name: "all crons should have non-empty help",
			test: func(t *testing.T) {
				for _, r := range cronRules {
					assert.NotEmpty(t, r.Help, "help for %q should not be empty", r.Name)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}
