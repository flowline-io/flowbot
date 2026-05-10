package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Count(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "five command rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, commandRules, 5)
		})
	}
}

func TestCommandRules_Defines(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all expected webhook commands defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "webhook list")
			assert.Contains(t, defines, "webhook create [flag]")
			assert.Contains(t, defines, "webhook del [secret]")
			assert.Contains(t, defines, "webhook activate [secret]")
			assert.Contains(t, defines, "webhook inactive [secret]")
		})
	}
}

func TestCommandRules_Handlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{
			name:   "webhook list exact match",
			define: "webhook list",
			input:  "webhook list",
			want:   true,
		},
		{
			name:   "webhook create with bracketed flag",
			define: "webhook create [flag]",
			input:  "webhook create [flag]",
			want:   true,
		},
		{
			name:   "webhook create with real flag value",
			define: "webhook create [flag]",
			input:  "webhook create myflag",
			want:   true,
		},
		{
			name:   "webhook del with bracketed secret",
			define: "webhook del [secret]",
			input:  "webhook del [secret]",
			want:   true,
		},
		{
			name:   "webhook del with real secret value",
			define: "webhook del [secret]",
			input:  "webhook del mysecret",
			want:   true,
		},
		{
			name:   "webhook activate with bracketed secret",
			define: "webhook activate [secret]",
			input:  "webhook activate [secret]",
			want:   true,
		},
		{
			name:   "webhook activate with real secret value",
			define: "webhook activate [secret]",
			input:  "webhook activate mysecret",
			want:   true,
		},
		{
			name:   "webhook inactive with bracketed secret",
			define: "webhook inactive [secret]",
			input:  "webhook inactive [secret]",
			want:   true,
		},
		{
			name:   "webhook inactive with real secret value",
			define: "webhook inactive [secret]",
			input:  "webhook inactive mysecret",
			want:   true,
		},
		{
			name:   "webhook list mismatched with create input",
			define: "webhook list",
			input:  "webhook create [flag]",
			want:   false,
		},
		{
			name:   "webhook create mismatched with list input",
			define: "webhook create [flag]",
			input:  "webhook list",
			want:   false,
		},
		{
			name:   "webhook list with extra tokens fails",
			define: "webhook list",
			input:  "webhook list extra",
			want:   false,
		},
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
		{name: "unknown command returns nil result"},
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

func TestCommandRules_ListHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "list handler returns info message with non-empty title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestCommandRules_CreateHandler_MissingFlag(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "create handler with missing flag returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestCommandRules_DelHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "del handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestCommandRules_ActivateHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "activate handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestCommandRules_InactiveHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "inactive handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
