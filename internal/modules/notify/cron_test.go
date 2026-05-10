package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

func TestCronRules(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "count",
			fn: func(t *testing.T) {
				assert.Len(t, cronRules, 1)
			},
		},
		{
			name: "name is notify_example",
			fn: func(t *testing.T) {
				assert.Equal(t, "notify_example", cronRules[0].Name)
			},
		},
		{
			name: "scope is system",
			fn: func(t *testing.T) {
				assert.Equal(t, cron.CronScopeSystem, cronRules[0].Scope)
			},
		},
		{
			name: "when is every minute",
			fn: func(t *testing.T) {
				assert.Equal(t, "* * * * *", cronRules[0].When)
			},
		},
		{
			name: "action not nil",
			fn: func(t *testing.T) {
				assert.NotNil(t, cronRules[0].Action)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}
