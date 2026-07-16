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
	t.Parallel()
	testRules := []Rule{
		{
			Define: `test`,
			Help:   `Test info`,
			Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
				return types.TextMsg{Text: "test"}
			},
		},
		{
			Define: `todo [string]`,
			Help:   `todo something`,
			Handler: func(_ types.Context, tokens []*parser.Token) types.MsgPayload {
				text, _ := tokens[1].Value.String()
				return types.TextMsg{Text: text}
			},
		},
		{
			Define: `add [number] [number]`,
			Help:   `Addition`,
			Handler: func(_ types.Context, tokens []*parser.Token) types.MsgPayload {
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
			t.Parallel()
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
	t.Parallel()
	testRules := []Rule{
		{
			Define: `test`,
			Help:   `Test info`,
			Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
				return types.TextMsg{Text: "test"}
			},
		},
	}

	b := Ruleset(testRules)

	t.Run("help returns non-nil", func(t *testing.T) {
		t.Parallel()
		help, err := b.Help("help")
		require.NoError(t, err)
		assert.NotNil(t, help)
	})
}

func TestRuleMetadata(t *testing.T) {
	t.Parallel()
	rule := Rule{Define: "ping", Help: "Ping help"}
	tests := []struct {
		name string
		want any
		got  func() any
	}{
		{name: "ID returns define", got: func() any { return rule.ID() }, want: "ping"},
		{name: "TYPE returns command rule", got: func() any { return rule.TYPE() }, want: types.CommandRule},
		{name: "TYPE is stable across calls", got: func() any { return Rule{}.TYPE() }, want: types.CommandRule},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got())
		})
	}
}

func TestHelpVariants(t *testing.T) {
	t.Parallel()
	rs := Ruleset([]Rule{
		{Define: "alpha", Help: "Alpha help", Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "a"}
		}},
		{Define: "beta", Help: "Beta help", Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "b"}
		}},
	})

	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantTitle string
	}{
		{name: "short h alias returns help", input: "h", wantTitle: "Help"},
		{name: "uppercase HELP returns help", input: "HELP", wantTitle: "Help"},
		{name: "non-help input returns nil", input: "alpha", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := rs.Help(tt.input)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			info, ok := got.(types.InfoMsg)
			require.True(t, ok)
			assert.Equal(t, tt.wantTitle, info.Title)
			assert.NotEmpty(t, info.Model)
		})
	}
}

func TestProcessCommandNoMatch(t *testing.T) {
	t.Parallel()
	rs := Ruleset([]Rule{
		{Define: "known", Help: "Known", Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "ok"}
		}},
	})

	tests := []struct {
		name    string
		command string
	}{
		{name: "unknown command returns nil payload", command: "missing"},
		{name: "partial token mismatch returns nil", command: "know"},
		{name: "empty command returns nil", command: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := rs.ProcessCommand(types.Context{}, tt.command)
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}
