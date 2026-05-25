package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestReaderCommandRules_Count(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "command rules should not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, commandRules)
		})
	}
}

func TestReaderCommandRules_Defines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "reader command defined with help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}
			assert.Contains(t, defines, "reader")
			assert.Equal(t, "show reader id", defines["reader"])
		})
	}
}

func TestReaderCommandRules_Handlers(t *testing.T) {
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

func TestReaderCommandRules_TokenParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{
			name:   "reader exact match",
			define: "reader",
			input:  "reader",
			want:   true,
		},
		{
			name:   "reader with extra tokens fails",
			define: "reader",
			input:  "reader extra",
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

func TestReaderCommandRules_ProcessCommand_Unknown(t *testing.T) {
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

func TestReaderCommandRules_ReaderHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "reader handler returns miniflux id as text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var readerRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "reader" {
					readerRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, readerRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("reader")

			payload := readerRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Equal(t, miniflux.ID, msg.Text)
		})
	}
}

func TestGetAIResult_FunctionExists(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "getAIResult function exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, getAIResult)
		})
	}
}
