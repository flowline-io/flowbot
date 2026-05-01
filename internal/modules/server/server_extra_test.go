package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_VersionHandler_Content(t *testing.T) {
	var versionRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "version" {
			versionRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, versionRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("version")

	payload := versionRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_MemStatsHandler_Content(t *testing.T) {
	var memRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "mem stats" {
			memRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, memRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("mem stats")

	payload := memRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.InfoMsg)
	require.True(t, ok)
	assert.Equal(t, "Memory stats", msg.Title)
}

func TestCommandRules_GolangStatsHandler_Content(t *testing.T) {
	var golangRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "golang stats" {
			golangRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, golangRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("golang stats")

	payload := golangRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.InfoMsg)
	require.True(t, ok)
	assert.Equal(t, "Golang stats", msg.Title)
}

func TestCommandRules_QueueStatsHandler_Content(t *testing.T) {
	var queueRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "queue stats" {
			queueRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, queueRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("queue stats")

	payload := queueRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.LinkMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Url, "/queue/stats")
	assert.Equal(t, "Queue Stats", msg.Title)
}

func TestCronNames(t *testing.T) {
	expected := []string{
		"server_user_online_change",
		"docker_images_prune",
		"docker_metrics",
		"monitor_metrics",
		"rules_updater",
		"online_agent_checker",
	}
	names := make(map[string]bool)
	for _, r := range cronRules {
		names[r.Name] = true
	}
	for _, name := range expected {
		assert.True(t, names[name], "cron rule %q should be defined", name)
	}
}

func TestCronScopes(t *testing.T) {
	scopeMap := make(map[string]string)
	for _, r := range cronRules {
		scopeMap[r.Name] = string(r.Scope)
	}
	assert.Equal(t, "user", scopeMap["server_user_online_change"])
	assert.Equal(t, "system", scopeMap["docker_images_prune"])
	assert.Equal(t, "system", scopeMap["docker_metrics"])
	assert.Equal(t, "system", scopeMap["monitor_metrics"])
	assert.Equal(t, "system", scopeMap["rules_updater"])
	assert.Equal(t, "system", scopeMap["online_agent_checker"])
}

func TestCronActionsNotNil(t *testing.T) {
	for _, r := range cronRules {
		assert.NotNil(t, r.Action, "action for %q should not be nil", r.Name)
	}
}

func TestCronWhenPatterns(t *testing.T) {
	whenMap := make(map[string]string)
	for _, r := range cronRules {
		whenMap[r.Name] = r.When
	}
	assert.Equal(t, "* * * * *", whenMap["server_user_online_change"])
	assert.Equal(t, "0 4 * * *", whenMap["docker_images_prune"])
	assert.Equal(t, "* * * * *", whenMap["docker_metrics"])
	assert.Equal(t, "* * * * *", whenMap["monitor_metrics"])
	assert.Equal(t, "*/2 * * * *", whenMap["online_agent_checker"])
}

func TestWebserviceRulesEndpoints(t *testing.T) {
	assert.GreaterOrEqual(t, len(webserviceRules), 2)
}
