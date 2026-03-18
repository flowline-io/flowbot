package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 10)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "version")
	assert.Contains(t, defines, "mem stats")
	assert.Contains(t, defines, "golang stats")
	assert.Contains(t, defines, "server stats")
	assert.Contains(t, defines, "online stats")
	assert.Contains(t, defines, "instruct list")
	assert.Contains(t, defines, "adguard status")
	assert.Contains(t, defines, "adguard stats")
	assert.Contains(t, defines, "queue stats")
	assert.Contains(t, defines, "check")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		define string
		input  string
		want   bool
	}{
		{"version", "version", true},
		{"mem stats", "mem stats", true},
		{"golang stats", "golang stats", true},
		{"server stats", "server stats", true},
		{"online stats", "online stats", true},
		{"instruct list", "instruct list", true},
		{"adguard status", "adguard status", true},
		{"adguard stats", "adguard stats", true},
		{"queue stats", "queue stats", true},
		{"check", "check", true},
		{"version", "mem stats", false},
		{"mem stats", "version", false},
		{"version", "version extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.define+"_"+tt.input, func(t *testing.T) {
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	rs := command.Ruleset(commandRules)
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

	result, err := rs.ProcessCommand(ctx, "unknown command xyz")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCommandRules_VersionHandler(t *testing.T) {
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

func TestCommandRules_MemStatsHandler(t *testing.T) {
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
	assert.NotEmpty(t, msg.Title)
}

func TestCommandRules_GolangStatsHandler(t *testing.T) {
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
	assert.NotEmpty(t, msg.Title)
}

func TestCommandRules_ServerStatsHandler(t *testing.T) {
	t.Skip("requires database connection")

	var serverRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "server stats" {
			serverRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, serverRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("server stats")

	payload := serverRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	assert.Contains(t, []string{"TextMsg", "InfoMsg"}, msgType)
}

func TestCommandRules_OnlineStatsHandler(t *testing.T) {
	t.Skip("requires redis service")

	var onlineRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "online stats" {
			onlineRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, onlineRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("online stats")

	payload := onlineRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	assert.Contains(t, []string{"TextMsg", "KVMsg"}, msgType)
}

func TestCommandRules_InstructListHandler(t *testing.T) {
	var instructRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "instruct list" {
			instructRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, instructRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("instruct list")

	payload := instructRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.InfoMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Title)
}

func TestCommandRules_AdguardStatusHandler(t *testing.T) {
	t.Skip("requires adguard service")

	var adguardRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "adguard status" {
			adguardRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, adguardRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("adguard status")

	payload := adguardRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_QueueStatsHandler(t *testing.T) {
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
}

func TestCommandRules_CheckHandler(t *testing.T) {
	t.Skip("requires external services")

	var checkRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "check" {
			checkRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, checkRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("check")

	payload := checkRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}
