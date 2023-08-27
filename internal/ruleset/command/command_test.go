package command

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
	"strconv"
	"testing"

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

	out, err := b.ProcessCommand(types.Context{}, "test")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, out, types.TextMsg{Text: "test"})

	out2, err := b.ProcessCommand(types.Context{}, "add 1 2")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, out2, types.TextMsg{Text: "3"})

	out3, err := b.ProcessCommand(types.Context{}, "help")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, out3 == nil)

	help, err := b.Help("help")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, help != nil)

	out4, err := b.ProcessCommand(types.Context{}, `todo "a b c"`)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, out4, types.TextMsg{Text: "a b c"})
}
