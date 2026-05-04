package rules

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"*", "server.offline", true},
		{"bookmark.created", "bookmark.created", true},
		{"bookmark.created", "bookmark.archived", false},
		{"infra.*", "infra.host.down", true},
		{"infra.*", "infra.host.up", true},
		{"infra.*", "bookmark.created", false},
		{"infra.*", "infra", false},
		{"*.created", "bookmark.created", true},
		{"*.created", "kanban.task.created", true},
		{"*.created", "server.offline", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEngineEvaluate(t *testing.T) {
	e := New(nil)
	err := e.LoadConfig([]config.NotifyRule{
		{
			ID:       "night_mute",
			Action:   config.NotifyRuleActionMute,
			Match:    config.NotifyRuleMatch{Event: "*", Channel: "*"},
			Condition: "time.hour >= 25",
			Priority:  100,
		},
		{
			ID:      "infra_throttle",
			Action:  config.NotifyRuleActionThrottle,
			Match:   config.NotifyRuleMatch{Event: "infra.*", Channel: "*"},
			Priority: 50,
			Params: config.NotifyRuleParams{Window: "5m", Limit: 1},
		},
		{
			ID:      "drop_rule",
			Action:  config.NotifyRuleActionDrop,
			Match:   config.NotifyRuleMatch{Event: "test.drop", Channel: "*"},
			Priority: 10,
		},
	})
	require.NoError(t, err)

	// night_mute rule has condition that does not match current hour
	// so infra throttle rule should match for infra.* events
	result := e.Evaluate(nil, "infra.host.down", "slack")
	require.NotNil(t, result)
	assert.Equal(t, config.NotifyRuleActionThrottle, result.Action)
	assert.Equal(t, "5m", result.Window)
	assert.Equal(t, 1, result.Limit)
	assert.False(t, result.Muted)

	// test drop rule
	result = e.Evaluate(nil, "test.drop", "slack")
	require.NotNil(t, result)
	assert.Equal(t, config.NotifyRuleActionDrop, result.Action)

	// no match for unlisted events
	result = e.Evaluate(nil, "bookmark.created", "slack")
	assert.Nil(t, result)
}

func TestEngineEvaluateNoMatch(t *testing.T) {
	e := New(nil)
	err := e.LoadConfig([]config.NotifyRule{
		{
			ID:       "specific_rule",
			Action:   config.NotifyRuleActionDrop,
			Match:    config.NotifyRuleMatch{Event: "specific.event", Channel: "slack"},
			Priority: 10,
		},
	})
	require.NoError(t, err)

	result := e.Evaluate(nil, "other.event", "ntfy")
	assert.Nil(t, result)
}

func TestEngineEvaluateChannelMatch(t *testing.T) {
	e := New(nil)
	err := e.LoadConfig([]config.NotifyRule{
		{
			ID:      "slack_only",
			Action:  config.NotifyRuleActionDrop,
			Match:   config.NotifyRuleMatch{Event: "*", Channel: "slack"},
			Priority: 10,
		},
	})
	require.NoError(t, err)

	// matches slack
	result := e.Evaluate(nil, "any.event", "slack")
	require.NotNil(t, result)

	// doesn't match ntfy
	result = e.Evaluate(nil, "any.event", "ntfy")
	assert.Nil(t, result)
}

func TestParseHour(t *testing.T) {
	assert.Equal(t, 23, parseHour("23"))
	assert.Equal(t, 8, parseHour("8"))
	assert.Equal(t, 0, parseHour("0"))
}

func TestEvalTimeCondition(t *testing.T) {
	// save and restore currentHour
	origHour := currentHour
	currentHour = func() int { return 14 } // 2pm
	defer func() { currentHour = origHour }()

	assert.True(t, evalTimeCondition("time.hour >= 10"))
	assert.False(t, evalTimeCondition("time.hour >= 20"))
	assert.True(t, evalTimeCondition("time.hour < 16"))
	assert.False(t, evalTimeCondition("time.hour < 10"))
	assert.True(t, evalTimeCondition("time.hour == 14"))
	assert.False(t, evalTimeCondition("time.hour == 15"))
}
