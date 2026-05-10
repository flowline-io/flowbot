package workflow

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
		{name: "nine command rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, commandRules, 9)
		})
	}
}

func TestCommandRules_Defines(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all expected workflow commands defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			name:   "workflow list exact match",
			define: "workflow list",
			input:  "workflow list",
			want:   true,
		},
		{
			name:   "workflow get with id value",
			define: "workflow get [id]",
			input:  "workflow get abc123",
			want:   true,
		},
		{
			name:   "workflow create with name value",
			define: "workflow create [name]",
			input:  "workflow create myworkflow",
			want:   true,
		},
		{
			name:   "workflow update with id and name",
			define: "workflow update [id] [name]",
			input:  "workflow update abc123 newname",
			want:   true,
		},
		{
			name:   "workflow delete with id",
			define: "workflow delete [id]",
			input:  "workflow delete abc123",
			want:   true,
		},
		{
			name:   "workflow activate with id",
			define: "workflow activate [id]",
			input:  "workflow activate abc123",
			want:   true,
		},
		{
			name:   "workflow deactivate with id",
			define: "workflow deactivate [id]",
			input:  "workflow deactivate abc123",
			want:   true,
		},
		{
			name:   "workflow execute with id",
			define: "workflow execute [id]",
			input:  "workflow execute abc123",
			want:   true,
		},
		{
			name:   "workflow stat exact match",
			define: "workflow stat",
			input:  "workflow stat",
			want:   true,
		},
		{
			name:   "workflow list mismatched with get input",
			define: "workflow list",
			input:  "workflow get abc123",
			want:   false,
		},
		{
			name:   "workflow get mismatched with list input",
			define: "workflow get [id]",
			input:  "workflow list",
			want:   false,
		},
		{
			name:   "workflow get missing id argument",
			define: "workflow get [id]",
			input:  "workflow get",
			want:   false,
		},
		{
			name:   "workflow create missing name argument",
			define: "workflow create [name]",
			input:  "workflow create",
			want:   false,
		},
		{
			name:   "workflow update missing second argument",
			define: "workflow update [id] [name]",
			input:  "workflow update abc123",
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

func TestCommandRules_ProcessCommand_WorkflowList(t *testing.T) {
	t.Skip("requires n8n service")

	tests := []struct {
		name string
	}{
		{name: "workflow list returns text or empty message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "workflow list")
			require.NoError(t, err)
			require.NotNil(t, result)

			msgType := types.TypeOf(result)
			assert.Contains(t, []string{"TextMsg", "EmptyMsg"}, msgType)
		})
	}
}

func TestCommandRules_ProcessCommand_WorkflowStat(t *testing.T) {
	t.Skip("requires n8n service")

	tests := []struct {
		name string
	}{
		{name: "workflow stat returns text or empty message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "workflow stat")
			require.NoError(t, err)
			require.NotNil(t, result)

			msgType := types.TypeOf(result)
			assert.Contains(t, []string{"TextMsg", "EmptyMsg"}, msgType)
		})
	}
}

func TestCommandRules_MissingParameters(t *testing.T) {
	tests := []struct {
		name   string
		define string
	}{
		{
			name:   "workflow get handler with missing id returns usage text",
			define: "workflow get [id]",
		},
		{
			name:   "workflow create handler with missing name returns usage text",
			define: "workflow create [name]",
		},
		{
			name:   "workflow update handler with missing params returns usage text",
			define: "workflow update [id] [name]",
		},
		{
			name:   "workflow delete handler with missing id returns usage text",
			define: "workflow delete [id]",
		},
		{
			name:   "workflow activate handler with missing id returns usage text",
			define: "workflow activate [id]",
		},
		{
			name:   "workflow deactivate handler with missing id returns usage text",
			define: "workflow deactivate [id]",
		},
		{
			name:   "workflow execute handler with missing id returns usage text",
			define: "workflow execute [id]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == tt.define {
					rule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, rule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			var manualTokens []*parser.Token
			switch tt.define {
			case "workflow get [id]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("get")},
				}
			case "workflow create [name]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("create")},
				}
			case "workflow update [id] [name]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("update")},
				}
			case "workflow delete [id]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("delete")},
				}
			case "workflow activate [id]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("activate")},
				}
			case "workflow deactivate [id]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("deactivate")},
				}
			case "workflow execute [id]":
				manualTokens = []*parser.Token{
					{Type: "character", Value: parser.Variable("workflow")},
					{Type: "character", Value: parser.Variable("execute")},
				}
			}

			payload := rule.Handler(ctx, manualTokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, "Usage:")
		})
	}
}
