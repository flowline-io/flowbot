package command

import (
	"strconv"
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexRule(t *testing.T) {
	testRules := []Rule{
		{
			Define: `test`,
			Help:   `Test info`,
			Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
				return types.TextMsg{Text: "test"}
			},
		},
		{
			Define: `todo [string]`,
			Help:   `todo something`,
			Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
				text, _ := tokens[1].Value.String()
				return types.TextMsg{Text: text}
			},
		},
		{
			Define: `add [number] [number]`,
			Help:   `Addition`,
			Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
				tt1, _ := tokens[1].Value.Int64()
				tt2, _ := tokens[2].Value.Int64()
				return types.TextMsg{Text: strconv.Itoa(int(tt1 + tt2))}
			},
		},
	}

	b := Ruleset(testRules)

	tests := []struct {
		name    string
		command string
		want    types.MsgPayload
		wantErr bool
	}{
		{
			name:    "simple test command",
			command: "test",
			want:    types.TextMsg{Text: "test"},
			wantErr: false,
		},
		{
			name:    "add two numbers",
			command: "add 1 2",
			want:    types.TextMsg{Text: "3"},
			wantErr: false,
		},
		{
			name:    "help returns nil",
			command: "help",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "todo with quoted string",
			command: `todo "a b c"`,
			want:    types.TextMsg{Text: "a b c"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := b.ProcessCommand(types.Context{}, tt.command)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, out)
		})
	}
}

func TestHelp(t *testing.T) {
	testRules := []Rule{
		{
			Define: `test`,
			Help:   `Test info`,
			Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
				return types.TextMsg{Text: "test"}
			},
		},
	}

	b := Ruleset(testRules)

	t.Run("help returns non-nil", func(t *testing.T) {
		help, err := b.Help("help")
		require.NoError(t, err)
		assert.True(t, help != nil)
	})
}
