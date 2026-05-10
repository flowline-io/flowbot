package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

func TestCronRules_Count(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "two cron rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Len(t, cronRules, 2)
		})
	}
}

func TestCronRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "names are reader_metrics and reader_daily_summary",
			fn: func(t *testing.T) {
				names := make(map[string]bool)
				for _, r := range cronRules {
					names[r.Name] = true
				}

				assert.True(t, names["reader_metrics"])
				assert.True(t, names["reader_daily_summary"])
			},
		},
		{
			name: "all scopes are system",
			fn: func(t *testing.T) {
				for _, r := range cronRules {
					assert.Equal(t, cron.CronScopeSystem, r.Scope)
				}
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
