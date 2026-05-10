package notify

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
		{name: "three command rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, commandRules, 3)
		})
	}
}

func TestCommandRules_Defines(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "notify list, delete, and config defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "notify list")
			assert.Contains(t, defines, "notify delete [string]")
			assert.Contains(t, defines, "notify config")
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
			name:   "notify list exact match",
			define: "notify list",
			input:  "notify list",
			want:   true,
		},
		{
			name:   "notify delete with bracketed arg",
			define: "notify delete [string]",
			input:  "notify delete [name]",
			want:   true,
		},
		{
			name:   "notify delete with real value",
			define: "notify delete [string]",
			input:  "notify delete mynotify",
			want:   true,
		},
		{
			name:   "notify config exact match",
			define: "notify config",
			input:  "notify config",
			want:   true,
		},
		{
			name:   "notify list mismatched with delete input",
			define: "notify list",
			input:  "notify delete [name]",
			want:   false,
		},
		{
			name:   "notify delete mismatched with list input",
			define: "notify delete [string]",
			input:  "notify list",
			want:   false,
		},
		{
			name:   "notify config mismatched with list input",
			define: "notify config",
			input:  "notify list",
			want:   false,
		},
		{
			name:   "notify delete missing argument",
			define: "notify delete [string]",
			input:  "notify delete",
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
		{name: "list handler returns info message with title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var listRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "notify list" {
					listRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, listRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("notify list")

			payload := listRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Title)
		})
	}
}

func TestCommandRules_ConfigHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "config handler returns valid message type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "notify config" {
					configRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, configRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("notify config")

			payload := configRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"LinkMsg", "FormMsg", "TextMsg"}, msgType)
		})
	}
}
