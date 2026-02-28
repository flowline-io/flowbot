package reader

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/stretchr/testify/assert"
)

func TestCronRules_Count(t *testing.T) {
	assert.Len(t, cronRules, 2)
}

func TestCronRules_Names(t *testing.T) {
	names := make(map[string]bool)
	for _, r := range cronRules {
		names[r.Name] = true
	}

	assert.True(t, names["reader_metrics"])
	assert.True(t, names["reader_daily_summary"])
}

func TestCronRules_Scopes(t *testing.T) {
	for _, r := range cronRules {
		assert.Equal(t, cron.CronScopeSystem, r.Scope)
	}
}

func TestCronRules_Actions(t *testing.T) {
	for _, r := range cronRules {
		assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
	}
}
