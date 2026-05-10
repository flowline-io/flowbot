package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyntax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantValues []string
	}{
		{
			name:       "parses command with typed parameters",
			input:      "subs open [string] [number] [any] [bool]",
			wantValues: []string{"subs", "open", "string", "number", "any", "bool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewSyntax([]rune(tt.input))
			for i, wantVal := range tt.wantValues {
				token, err := s.GetNextToken()
				require.NoError(t, err, "token %d", i)
				assert.Equal(t, Variable(wantVal), token.Value, "token %d", i)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	define := "subs open [string] [number] [any] [bool]"
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{name: "valid args pass check", define: define, input: "subs open abc 123 demo true", want: true},
		{name: "non-numeric where number expected fails", define: define, input: "subs open abc no_num demo true", want: false},
		{name: "non-boolean where bool expected fails", define: define, input: "subs open abc 123 demo t", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, err := ParseString(tt.input)
			require.NoError(t, err)
			c, err := SyntaxCheck(tt.define, a)
			require.NoError(t, err)
			assert.Equal(t, tt.want, c)
		})
	}
}
