package search

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
	assert.Equal(t, "search", commandRules[0].Define)
	assert.Equal(t, "Search page", commandRules[0].Help)
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
		{"search", "search", true},
		{"search", "search extra", false},
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

func TestCommandRules_SearchHandler(t *testing.T) {
	t.Skip("requires database connection")

	var searchRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "search" {
			searchRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, searchRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("search")

	payload := searchRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.LinkMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Url, "/page/search/")
}
