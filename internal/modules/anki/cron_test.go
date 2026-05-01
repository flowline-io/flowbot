package anki

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/stretchr/testify/assert"
)

func TestCronRules_Count(t *testing.T) {
	assert.Len(t, cronRules, 1)
}

func TestCronRules_Name(t *testing.T) {
	assert.Equal(t, "anki_review_remind", cronRules[0].Name)
}

func TestCronRules_Scope(t *testing.T) {
	assert.Equal(t, cron.CronScopeUser, cronRules[0].Scope)
}

func TestCronRules_When(t *testing.T) {
	assert.Equal(t, "* * * * *", cronRules[0].When)
}

func TestCronRules_Actions(t *testing.T) {
	for _, r := range cronRules {
		assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
	}
}
