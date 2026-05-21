package example

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
		{name: "should have exactly 4 command rules", test: func(t *testing.T) {
			t.Parallel()
			assert.Len(t, commandRules, 4)
		}},
		{name: "should contain all expected defines", test: func(t *testing.T) {
			t.Parallel()
			expected := []string{"id", "form test", "page test", "event test"}
			defines := make(map[string]bool)
			for _, r := range commandRules {
				defines[r.Define] = true
			}
			for _, e := range expected {
				assert.True(t, defines[e], "expected define %q to exist", e)
			}
		}},
		{name: "all command rules should have non-nil handlers", test: func(t *testing.T) {
			t.Parallel()
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		}},
		{name: "all command rules should have non-empty help", test: func(t *testing.T) {
			t.Parallel()
			for _, r := range commandRules {
				assert.NotEmpty(t, r.Help, "help for %q should not be empty", r.Define)
			}
		}},
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
		{name: "id exact match", define: "id", input: "id", want: true},
		{name: "form test exact match", define: "form test", input: "form test", want: true},
		{name: "page test exact match", define: "page test", input: "page test", want: true},
		{name: "event test exact match", define: "event test", input: "event test", want: true},
		{name: "id does not match form test", define: "id", input: "form test", want: false},
		{name: "form test with extra tokens", define: "form test", input: "form test extra", want: false},
		{name: "id with extra tokens", define: "id", input: "id extra", want: false},
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

func TestCommandRules_IDHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should generate unique non-empty UUIDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var idRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "id" {
					idRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, idRule)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("id")
			results := make(map[string]bool)
			for range 10 {
				payload := idRule.Handler(ctx, tokens)
				require.NotNil(t, payload)
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok, "should be TextMsg")
				assert.NotEmpty(t, msg.Text, "ID should not be empty")
				assert.Greater(t, len(msg.Text), 10, "ID should be reasonably long")
				results[msg.Text] = true
			}
			assert.GreaterOrEqual(t, len(results), 2, "should generate different UUIDs")
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
