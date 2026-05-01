package webhook

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 5)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "webhook list")
	assert.Contains(t, defines, "webhook create [flag]")
	assert.Contains(t, defines, "webhook del [secret]")
	assert.Contains(t, defines, "webhook activate [secret]")
	assert.Contains(t, defines, "webhook inactive [secret]")
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
		{"webhook list", "webhook list", true},
		{"webhook create [flag]", "webhook create [flag]", true},
		{"webhook create [flag]", "webhook create myflag", true},
		{"webhook del [secret]", "webhook del [secret]", true},
		{"webhook del [secret]", "webhook del mysecret", true},
		{"webhook activate [secret]", "webhook activate [secret]", true},
		{"webhook activate [secret]", "webhook activate mysecret", true},
		{"webhook inactive [secret]", "webhook inactive [secret]", true},
		{"webhook inactive [secret]", "webhook inactive mysecret", true},
		{"webhook list", "webhook create [flag]", false},
		{"webhook create [flag]", "webhook list", false},
		{"webhook list", "webhook list extra", false},
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

func TestCommandRules_ListHandler(t *testing.T) {
	t.Skip("requires database connection")

	var listRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "webhook list" {
			listRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, listRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("webhook list")

	payload := listRule.Handler(ctx, tokens)
	if payload != nil {
		msg, ok := payload.(types.InfoMsg)
		require.True(t, ok)
		assert.NotEmpty(t, msg.Title)
	}
}

func TestCommandRules_CreateHandler_MissingFlag(t *testing.T) {
	t.Skip("requires database connection")

	var createRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "webhook create [flag]" {
			createRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, createRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("webhook")},
		{Type: "character", Value: parser.Variable("create")},
	}

	payload := createRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_DelHandler(t *testing.T) {
	t.Skip("requires database connection")

	var delRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "webhook del [secret]" {
			delRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, delRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("webhook del [secret]")

	payload := delRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_ActivateHandler(t *testing.T) {
	t.Skip("requires database connection")

	var activateRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "webhook activate [secret]" {
			activateRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, activateRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("webhook activate [secret]")

	payload := activateRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_InactiveHandler(t *testing.T) {
	t.Skip("requires database connection")

	var inactiveRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "webhook inactive [secret]" {
			inactiveRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, inactiveRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("webhook inactive [secret]")

	payload := inactiveRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}
