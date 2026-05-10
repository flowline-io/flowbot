package github

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
			name: "should have exactly 2 cron rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, cronRules, 2)
			},
		},
		{
			name: "should contain expected cron names",
			test: func(t *testing.T) {
				t.Parallel()
				names := make(map[string]bool)
				for _, r := range cronRules {
					names[r.Name] = true
				}

				assert.True(t, names["github_starred"])
				assert.True(t, names["github_notifications"])
			},
		},
		{
			name: "all crons should have user scope",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range cronRules {
					assert.Equal(t, cron.CronScopeUser, r.Scope)
				}
			},
		},
		{
			name: "all crons should have non-nil actions",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range cronRules {
					assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
