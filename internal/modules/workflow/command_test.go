package workflow

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 9)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "workflow list")
	assert.Contains(t, defines, "workflow get [id]")
	assert.Contains(t, defines, "workflow create [name]")
	assert.Contains(t, defines, "workflow update [id] [name]")
	assert.Contains(t, defines, "workflow delete [id]")
	assert.Contains(t, defines, "workflow activate [id]")
	assert.Contains(t, defines, "workflow deactivate [id]")
	assert.Contains(t, defines, "workflow execute [id]")
	assert.Contains(t, defines, "workflow stat")
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
		{"workflow list", "workflow list", true},
		{"workflow get [id]", "workflow get abc123", true},
		{"workflow create [name]", "workflow create myworkflow", true},
		{"workflow update [id] [name]", "workflow update abc123 newname", true},
		{"workflow delete [id]", "workflow delete abc123", true},
		{"workflow activate [id]", "workflow activate abc123", true},
		{"workflow deactivate [id]", "workflow deactivate abc123", true},
		{"workflow execute [id]", "workflow execute abc123", true},
		{"workflow stat", "workflow stat", true},
		{"workflow list", "workflow get abc123", false},
		{"workflow get [id]", "workflow list", false},
		{"workflow get [id]", "workflow get", false},
		{"workflow create [name]", "workflow create", false},
		{"workflow update [id] [name]", "workflow update abc123", false},
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

func TestCommandRules_ProcessCommand_WorkflowList(t *testing.T) {
	t.Skip("requires n8n service")

	rs := command.Ruleset(commandRules)
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

	result, err := rs.ProcessCommand(ctx, "workflow list")
	require.NoError(t, err)
	require.NotNil(t, result)

	msgType := types.TypeOf(result)
	assert.Contains(t, []string{"TextMsg", "EmptyMsg"}, msgType)
}

func TestCommandRules_ProcessCommand_WorkflowStat(t *testing.T) {
	t.Skip("requires n8n service")

	rs := command.Ruleset(commandRules)
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

	result, err := rs.ProcessCommand(ctx, "workflow stat")
	require.NoError(t, err)
	require.NotNil(t, result)

	msgType := types.TypeOf(result)
	assert.Contains(t, []string{"TextMsg", "EmptyMsg"}, msgType)
}

func TestCommandRules_GetHandler_MissingId(t *testing.T) {
	var getRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow get [id]" {
			getRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, getRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("get")},
	}

	payload := getRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_CreateHandler_MissingName(t *testing.T) {
	var createRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow create [name]" {
			createRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, createRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("create")},
	}

	payload := createRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_UpdateHandler_MissingParams(t *testing.T) {
	var updateRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow update [id] [name]" {
			updateRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, updateRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("update")},
	}

	payload := updateRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_DeleteHandler_MissingId(t *testing.T) {
	var deleteRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow delete [id]" {
			deleteRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, deleteRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("delete")},
	}

	payload := deleteRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_ActivateHandler_MissingId(t *testing.T) {
	var activateRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow activate [id]" {
			activateRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, activateRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("activate")},
	}

	payload := activateRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_DeactivateHandler_MissingId(t *testing.T) {
	var deactivateRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow deactivate [id]" {
			deactivateRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, deactivateRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("deactivate")},
	}

	payload := deactivateRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}

func TestCommandRules_ExecuteHandler_MissingId(t *testing.T) {
	var executeRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "workflow execute [id]" {
			executeRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, executeRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens := []*parser.Token{
		{Type: "character", Value: parser.Variable("workflow")},
		{Type: "character", Value: parser.Variable("execute")},
	}

	payload := executeRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Usage:")
}
