package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

func TestMatchPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		{name: "star matches anything", pattern: "*", value: "anything", want: true},
		{name: "star matches dotted event", pattern: "*", value: "server.offline", want: true},
		{name: "exact match", pattern: "bookmark.created", value: "bookmark.created", want: true},
		{name: "exact mismatch", pattern: "bookmark.created", value: "bookmark.archived", want: false},
		{name: "prefix star matches host.down", pattern: "infra.*", value: "infra.host.down", want: true},
		{name: "prefix star matches host.up", pattern: "infra.*", value: "infra.host.up", want: true},
		{name: "prefix star does not match unrelated", pattern: "infra.*", value: "bookmark.created", want: false},
		{name: "prefix star does not match bare prefix", pattern: "infra.*", value: "infra", want: false},
		{name: "suffix star matches bookmark.created", pattern: "*.created", value: "bookmark.created", want: true},
		{name: "suffix star matches kanban.task.created", pattern: "*.created", value: "kanban.task.created", want: true},
		{name: "suffix star does not match different suffix", pattern: "*.created", value: "server.offline", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchPattern(tt.pattern, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEngineEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		rules      []manifest.Rule
		eventType  string
		channel    string
		wantAction manifest.RuleAction
		wantWindow string
		wantLimit  int
		wantMuted  bool
		wantNil    bool
	}{
		{
			name: "infra throttle rule matches infra.host.down",
			rules: []manifest.Rule{
				{
					ID:        "night_mute",
					Action:    manifest.RuleActionMute,
					Match:     manifest.RuleMatch{Event: "*", Channel: "*"},
					Condition: "time.hour >= 25",
					Priority:  100,
				},
				{
					ID:       "infra_throttle",
					Action:   manifest.RuleActionThrottle,
					Match:    manifest.RuleMatch{Event: "infra.*", Channel: "*"},
					Priority: 50,
					Params:   manifest.RuleParams{Window: "5m", Limit: 1},
				},
				{
					ID:       "drop_rule",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "test.drop", Channel: "*"},
					Priority: 10,
				},
			},
			eventType:  "infra.host.down",
			channel:    "slack",
			wantAction: manifest.RuleActionThrottle,
			wantWindow: "5m",
			wantLimit:  1,
			wantMuted:  false,
		},
		{
			name: "drop rule matches test.drop",
			rules: []manifest.Rule{
				{
					ID:        "night_mute",
					Action:    manifest.RuleActionMute,
					Match:     manifest.RuleMatch{Event: "*", Channel: "*"},
					Condition: "time.hour >= 25",
					Priority:  100,
				},
				{
					ID:       "infra_throttle",
					Action:   manifest.RuleActionThrottle,
					Match:    manifest.RuleMatch{Event: "infra.*", Channel: "*"},
					Priority: 50,
					Params:   manifest.RuleParams{Window: "5m", Limit: 1},
				},
				{
					ID:       "drop_rule",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "test.drop", Channel: "*"},
					Priority: 10,
				},
			},
			eventType:  "test.drop",
			channel:    "slack",
			wantAction: manifest.RuleActionDrop,
		},
		{
			name: "no match for unlisted event",
			rules: []manifest.Rule{
				{
					ID:        "night_mute",
					Action:    manifest.RuleActionMute,
					Match:     manifest.RuleMatch{Event: "*", Channel: "*"},
					Condition: "time.hour >= 25",
					Priority:  100,
				},
				{
					ID:       "infra_throttle",
					Action:   manifest.RuleActionThrottle,
					Match:    manifest.RuleMatch{Event: "infra.*", Channel: "*"},
					Priority: 50,
					Params:   manifest.RuleParams{Window: "5m", Limit: 1},
				},
				{
					ID:       "drop_rule",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "test.drop", Channel: "*"},
					Priority: 10,
				},
			},
			eventType: "bookmark.created",
			channel:   "slack",
			wantNil:   true,
		},
		{
			name: "no match when event and channel differ",
			rules: []manifest.Rule{
				{
					ID:       "specific_rule",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "specific.event", Channel: "slack"},
					Priority: 10,
				},
			},
			eventType: "other.event",
			channel:   "ntfy",
			wantNil:   true,
		},
		{
			name: "channel-specific rule matches correct channel",
			rules: []manifest.Rule{
				{
					ID:       "slack_only",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "*", Channel: "slack"},
					Priority: 10,
				},
			},
			eventType:  "any.event",
			channel:    "slack",
			wantAction: manifest.RuleActionDrop,
		},
		{
			name: "channel-specific rule does not match different channel",
			rules: []manifest.Rule{
				{
					ID:       "slack_only",
					Action:   manifest.RuleActionDrop,
					Match:    manifest.RuleMatch{Event: "*", Channel: "slack"},
					Priority: 10,
				},
			},
			eventType: "any.event",
			channel:   "ntfy",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New(nil)
			err := e.LoadConfig(tt.rules)
			require.NoError(t, err)

			result := e.Evaluate(nil, tt.eventType, tt.channel)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.wantAction, result.Action)
			if tt.wantWindow != "" {
				assert.Equal(t, tt.wantWindow, result.Window)
			}
			if tt.wantLimit != 0 {
				assert.Equal(t, tt.wantLimit, result.Limit)
			}
			assert.Equal(t, tt.wantMuted, result.Muted)
		})
	}
}

func TestParseHour(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "two-digit hour", input: "23", want: 23},
		{name: "single-digit hour", input: "8", want: 8},
		{name: "zero hour", input: "0", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseHour(tt.input))
		})
	}
}

func TestEvalTimeCondition(t *testing.T) {
	origHour := currentHour
	currentHour = func() int { return 14 }
	defer func() { currentHour = origHour }()

	tests := []struct {
		name      string
		condition string
		want      bool
	}{
		{name: "hour >= 10 when hour is 14", condition: "time.hour >= 10", want: true},
		{name: "hour >= 20 when hour is 14", condition: "time.hour >= 20", want: false},
		{name: "hour < 16 when hour is 14", condition: "time.hour < 16", want: true},
		{name: "hour < 10 when hour is 14", condition: "time.hour < 10", want: false},
		{name: "hour == 14 when hour is 14", condition: "time.hour == 14", want: true},
		{name: "hour == 15 when hour is 14", condition: "time.hour == 15", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, evalTimeCondition(tt.condition))
		})
	}
}
