package bookmark

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	assert.Equal(t, "bookmark", Name)
}

func TestBotInit_Enabled(t *testing.T) {
	handler = bot{} // reset
	config := configType{Enabled: true}
	data, _ := json.Marshal(config)
	err := handler.Init(data)
	require.NoError(t, err)
	assert.True(t, handler.IsReady())
}

func TestBotInit_Disabled(t *testing.T) {
	handler = bot{} // reset
	config := configType{Enabled: false}
	data, _ := json.Marshal(config)
	err := handler.Init(data)
	require.NoError(t, err)
	assert.False(t, handler.IsReady())
}

func TestBotInit_InvalidJSON(t *testing.T) {
	handler = bot{} // reset
	err := handler.Init(json.RawMessage(`{invalid`))
	assert.Error(t, err)
}

func TestBotInit_AlreadyInitialized(t *testing.T) {
	handler = bot{initialized: true}
	err := handler.Init(json.RawMessage(`{"enabled":true}`))
	assert.Error(t, err)
}

func TestCommandRules_Defined(t *testing.T) {
	assert.NotEmpty(t, commandRules)

	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "bookmark list")
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestCronRules_Defined(t *testing.T) {
	assert.NotEmpty(t, cronRules)

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

func TestCronRules_HaveActions(t *testing.T) {
	for _, r := range cronRules {
		assert.NotNil(t, r.Action, "action for cron %q should not be nil", r.Name)
		assert.NotEmpty(t, r.When, "when for cron %q should not be empty", r.Name)
	}
}

func TestEventRules_Defined(t *testing.T) {
	assert.NotEmpty(t, eventRules)

	ids := make(map[string]bool)
	for _, r := range eventRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[types.BookmarkArchiveBotEventID])
	assert.True(t, ids[types.BookmarkCreateBotEventID])
	assert.True(t, ids[types.ArchiveBoxAddBotEventID])
}

func TestEventRules_HaveHandlers(t *testing.T) {
	for _, r := range eventRules {
		assert.NotNil(t, r.Handler, "handler for event %q should not be nil", r.Id)
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	handler = bot{initialized: true}
	rules := handler.Rules()
	assert.NotEmpty(t, rules)
	assert.Len(t, rules, 3) // commandRules, cronRules, eventRules
}
