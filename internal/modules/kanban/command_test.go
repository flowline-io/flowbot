package kanban

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Count(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "has one command rule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Len(t, commandRules, 1)
		})
	}
}

func TestCommandRules_Defines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "kanban status defined with help text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "kanban status", commandRules[0].Define)
			assert.Equal(t, "Show kanban status", commandRules[0].Help)
		})
	}
}

func TestCommandRules_Handlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{
			name:   "kanban status exact match",
			define: "kanban status",
			input:  "kanban status",
			want:   true,
		},
		{
			name:   "kanban status with extra tokens",
			define: "kanban status",
			input:  "kanban status extra",
			want:   false,
		},
		{
			name:   "kanban partial match fails",
			define: "kanban status",
			input:  "kanban",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unknown command returns nil result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "unknown command xyz")
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestCommandRules_StatusHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "status handler returns empty message type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var statusRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "kanban status" {
					statusRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, statusRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("kanban status")

			payload := statusRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Equal(t, "EmptyMsg", msgType)
		})
	}
}
