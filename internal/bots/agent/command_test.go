package agent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 2)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "agent token")
	assert.Contains(t, defines, "agent reset token")
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
		{"agent token", "agent token", true},
		{"agent reset token", "agent reset token", true},
		{"agent token", "agent reset token", false},
		{"agent reset token", "agent token", false},
		{"agent token", "agent token extra", false},
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

func TestCommandRules_AgentTokenHandler(t *testing.T) {
	t.Skip("requires database connection")

	var tokenRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "agent token" {
			tokenRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, tokenRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("agent token")

	payload := tokenRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "token")
}

func TestCommandRules_AgentResetTokenHandler(t *testing.T) {
	t.Skip("requires database connection")

	var resetRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "agent reset token" {
			resetRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, resetRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("agent reset token")

	payload := resetRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "token")
}
