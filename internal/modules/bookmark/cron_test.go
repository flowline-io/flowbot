package bookmark

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
			name: "should have exactly 5 cron rules",
			test: func(t *testing.T) {
				assert.Len(t, cronRules, 5)
			},
		},
		{
			name: "should contain expected cron names",
			test: func(t *testing.T) {
				names := make(map[string]bool)
				for _, r := range cronRules {
					names[r.Name] = true
				}

				assert.True(t, names["bookmarks_tag"])
				assert.True(t, names["bookmarks_metrics"])
				assert.True(t, names["bookmarks_search"])
				assert.True(t, names["bookmarks_task"])
				assert.True(t, names["bookmarks_tag_merge"])
			},
		},
		{
			name: "should have correct scopes",
			test: func(t *testing.T) {
				scopeMap := make(map[string]cron.CronScope)
				for _, r := range cronRules {
					scopeMap[r.Name] = r.Scope
				}

				assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_tag"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_metrics"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_search"])
				assert.Equal(t, cron.CronScopeUser, scopeMap["bookmarks_task"])
				assert.Equal(t, cron.CronScopeSystem, scopeMap["bookmarks_tag_merge"])
			},
		},
		{
			name: "all cron rules should have non-nil actions",
			test: func(t *testing.T) {
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
