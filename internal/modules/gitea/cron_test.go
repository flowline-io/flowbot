package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

func TestCronRules(t *testing.T) {
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
			name: "should have name gitea_metrics",
			test: func(t *testing.T) {
				assert.Equal(t, "gitea_metrics", cronRules[0].Name)
			},
		},
		{
			name: "should have system scope",
			test: func(t *testing.T) {
				assert.Equal(t, cron.CronScopeSystem, cronRules[0].Scope)
			},
		},
		{
			name: "should have every minute cron expression",
			test: func(t *testing.T) {
				assert.Equal(t, "* * * * *", cronRules[0].When)
			},
		},
		{
			name: "should have non-nil action",
			test: func(t *testing.T) {
				assert.NotNil(t, cronRules[0].Action)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
