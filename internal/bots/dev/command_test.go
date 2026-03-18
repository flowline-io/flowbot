package dev

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 14)
}

func TestCommandRules_AllDefines(t *testing.T) {
	expected := []string{
		"dev setting", "id", "form test", "queue test",
		"instruct test", "page test", "docker test", "torrent test",
		"slash test", "llm test", "notify test", "fs test",
		"event test", "test",
	}
	defines := make(map[string]bool)
	for _, r := range commandRules {
		defines[r.Define] = true
	}
	for _, e := range expected {
		assert.True(t, defines[e], "expected define %q to exist", e)
	}
}

func TestCommandRules_AllHandlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestCommandRules_AllHaveHelp(t *testing.T) {
	for _, r := range commandRules {
		assert.NotEmpty(t, r.Help, "help for %q should not be empty", r.Define)
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		define string
		input  string
		want   bool
	}{
		{"id", "id", true},
		{"dev setting", "dev setting", true},
		{"form test", "form test", true},
		{"queue test", "queue test", true},
		{"instruct test", "instruct test", true},
		{"page test", "page test", true},
		{"docker test", "docker test", true},
		{"torrent test", "torrent test", true},
		{"slash test", "slash test", true},
		{"llm test", "llm test", true},
		{"notify test", "notify test", true},
		{"fs test", "fs test", true},
		{"event test", "event test", true},
		{"test", "test", true},
		{"id", "dev setting", false},
		{"dev setting", "id", false},
		{"form test", "form test extra", false},
		{"id", "id extra", false},
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

func TestCommandRules_IDHandler(t *testing.T) {
	var idRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "id" {
			idRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, idRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("id")

	results := make(map[string]bool)
	for i := 0; i < 10; i++ {
		payload := idRule.Handler(ctx, tokens)
		require.NotNil(t, payload)

		msg, ok := payload.(types.TextMsg)
		require.True(t, ok, "should be TextMsg")

		assert.NotEmpty(t, msg.Text, "ID should not be empty")
		assert.Greater(t, len(msg.Text), 10, "ID should be reasonably long")
		results[msg.Text] = true
	}

	assert.GreaterOrEqual(t, len(results), 2, "should generate different UUIDs")
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	rs := command.Ruleset(commandRules)
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

	result, err := rs.ProcessCommand(ctx, "unknown command xyz")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCommandRules_FormTestHandler(t *testing.T) {
	t.Skip("requires database connection")

	var formRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "form test" {
			formRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, formRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("form test")

	payload := formRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	if msgType == "FormMsg" {
		msg, ok := payload.(types.FormMsg)
		require.True(t, ok)
		assert.Equal(t, devFormID, msg.ID)
		assert.NotEmpty(t, msg.Title)
		assert.NotEmpty(t, msg.Field)
	} else {
		assert.Equal(t, "TextMsg", msgType)
	}
}

func TestCommandRules_DevSettingHandler(t *testing.T) {
	t.Skip("requires database connection")

	var settingRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "dev setting" {
			settingRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, settingRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("dev setting")

	payload := settingRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	if msgType == "LinkMsg" {
		msg, ok := payload.(types.LinkMsg)
		require.True(t, ok)
		assert.Contains(t, msg.Title, "Setting")
	} else {
		assert.Equal(t, "TextMsg", msgType)
	}
}

func TestCommandRules_PageTestHandler(t *testing.T) {
	t.Skip("requires database connection")

	var pageRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "page test" {
			pageRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, pageRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("page test")

	payload := pageRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	if msgType == "LinkMsg" {
		link, ok := payload.(types.LinkMsg)
		require.True(t, ok)
		assert.Contains(t, link.Url, "/page/dev/")
	} else {
		assert.Equal(t, "TextMsg", msgType)
	}
}

func TestCommandRules_SlashTestHandler(t *testing.T) {
	t.Skip("requires external service")

	var slashRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "slash test" {
			slashRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, slashRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("slash test")

	payload := slashRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}

func TestCommandRules_LLMTestHandler(t *testing.T) {
	t.Skip("requires agent service")

	var llmRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "llm test" {
			llmRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, llmRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("llm test")

	payload := llmRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.NotEmpty(t, msg.Text)
}
