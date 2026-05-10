package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Metadata(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 command rule",
			test: func(t *testing.T) {
				assert.Len(t, commandRules, 1)
			},
		},
		{
			name: "should have correct define and help",
			test: func(t *testing.T) {
				assert.Equal(t, "gitea", commandRules[0].Define)
				assert.Equal(t, "Example command", commandRules[0].Help)
			},
		},
		{
			name: "all command rules should have non-nil handlers",
			test: func(t *testing.T) {
				for _, r := range commandRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{name: "gitea exact match", define: "gitea", input: "gitea", want: true},
		{name: "gitea with extra tokens", define: "gitea", input: "gitea extra", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "unknown command should return nil result"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "unknown command xyz")
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestCommandRules_GiteaHandler(t *testing.T) {
	t.Skip("requires gitea service")

	tests := []struct {
		name string
	}{
		{name: "gitea handler should return non-empty TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var giteaRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "gitea" {
					giteaRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, giteaRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("gitea")

			payload := giteaRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}
