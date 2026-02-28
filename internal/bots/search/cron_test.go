package search

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/stretchr/testify/assert"
)

func TestCronRules_Count(t *testing.T) {
	assert.Len(t, cronRules, 1)
}

func TestCronRules_Name(t *testing.T) {
	assert.Equal(t, "search_example", cronRules[0].Name)
}

func TestCronRules_Scope(t *testing.T) {
	assert.Equal(t, cron.CronScopeSystem, cronRules[0].Scope)
}

func TestCronRules_When(t *testing.T) {
	assert.Equal(t, "* * * * *", cronRules[0].When)
}

func TestCronRules_Action(t *testing.T) {
	assert.NotNil(t, cronRules[0].Action)
}
