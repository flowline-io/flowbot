package cloudflare

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

	assert.Contains(t, defines, "cloudflare setting")
	assert.Contains(t, defines, "cloudflare test")
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
		{"cloudflare setting", "cloudflare setting", true},
		{"cloudflare test", "cloudflare test", true},
		{"cloudflare setting", "cloudflare test", false},
		{"cloudflare test", "cloudflare setting", false},
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

func TestCommandRules_SettingHandler(t *testing.T) {
	t.Skip("requires database connection")

	var settingRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "cloudflare setting" {
			settingRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, settingRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("cloudflare setting")

	payload := settingRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	assert.Contains(t, []string{"LinkMsg", "TextMsg"}, msgType)
}

func TestCommandRules_TestHandler_NoConfig(t *testing.T) {
	t.Skip("requires database connection")

	var testRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "cloudflare test" {
			testRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, testRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("cloudflare test")

	payload := testRule.Handler(ctx, tokens)
	if payload != nil {
		msg, ok := payload.(types.TextMsg)
		require.True(t, ok)
		assert.Contains(t, msg.Text, "config error")
	}
}
