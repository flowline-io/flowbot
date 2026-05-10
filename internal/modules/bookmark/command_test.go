package bookmark

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Metadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 command rule",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, commandRules, 1)
			},
		},
		{
			name: "should have correct define and help",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, "bookmark list", commandRules[0].Define)
				assert.Equal(t, "newest 10", commandRules[0].Help)
			},
		},
		{
			name: "all command rules should have non-nil handlers",
			test: func(t *testing.T) {
				t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{name: "exact match bookmark list", define: "bookmark list", input: "bookmark list", want: true},
		{name: "extra tokens should not match", define: "bookmark list", input: "bookmark list extra", want: false},
		{name: "partial match should not match", define: "bookmark list", input: "bookmark", want: false},
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
		{name: "unknown command should return nil result"},
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

func TestCommandRules_ListHandler(t *testing.T) {
	t.Skip("requires karakeep service")

	tests := []struct {
		name string
	}{
		{name: "list handler should return valid message type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var listRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "bookmark list" {
					listRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, listRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("bookmark list")

			payload := listRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "LinkMsg"}, msgType)
		})
	}
}
