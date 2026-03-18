package anki

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 1)
}

func TestCommandRules_Defines(t *testing.T) {
	assert.Equal(t, "stats", commandRules[0].Define)
	assert.Equal(t, "Anki collection statistics", commandRules[0].Help)
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
		{"stats", "stats", true},
		{"stats", "stats extra", false},
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

func TestCommandRules_StatsHandler(t *testing.T) {
	t.Skip("requires database connection")

	var statsRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "stats" {
			statsRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, statsRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("stats")

	payload := statsRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	assert.Contains(t, []string{"TextMsg", "LinkMsg", "HtmlMsg"}, msgType)
}
