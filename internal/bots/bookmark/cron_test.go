package bookmark

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/stretchr/testify/assert"
)

func TestCronRules_Count(t *testing.T) {
	assert.Len(t, cronRules, 5)
}

func TestCronRules_Names(t *testing.T) {
	names := make(map[string]bool)
	for _, r := range cronRules {
		names[r.Name] = true
	}

	assert.True(t, names["bookmarks_tag"])
	assert.True(t, names["bookmarks_metrics"])
	assert.True(t, names["bookmarks_search"])
	assert.True(t, names["bookmarks_task"])
	assert.True(t, names["bookmarks_tag_merge"])
}

func TestCronRules_Scopes(t *testing.T) {
	scopeMap := make(map[string]cron.CronScope)
	for _, r := range cronRules {
		scopeMap[r.Name] = r.Scope
	}

	assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_tag"])
	assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_metrics"])
	assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_search"])
	assert.Equal(t, cron.CronScopeUser, scopeMap["bookmarks_task"])
	assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_tag_merge"])
}

func TestCronRules_Actions(t *testing.T) {
	for _, r := range cronRules {
		assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
	}
}
