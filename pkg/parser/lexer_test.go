package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	t.Parallel()
	tokens := []string{CharacterToken, CharacterToken, CharacterToken, CharacterToken}

	tests := []struct {
		name      string
		input     string
		wantTypes []string
	}{
		{
			name:      "lexes simple command with quoted string",
			input:     "subs  open abc \"a b c\"",
			wantTypes: tokens,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := NewLexer([]rune(tt.input))
			for i, wantType := range tt.wantTypes {
				token, err := l.GetNextToken()
				require.NoError(t, err, "token %d", i)
				assert.Equal(t, wantType, token.Type, "token %d", i)
			}
			token, err := l.GetNextToken()
			require.NoError(t, err)
			assert.Equal(t, EOFToken, token.Type)
		})
	}
}

func TestParseCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantLen    int
		wantValues []string
	}{
		{
			name:       "single word command",
			input:      "subs",
			wantLen:    1,
			wantValues: []string{"subs"},
		},
		{
			name:       "two word command",
			input:      "subs list",
			wantLen:    2,
			wantValues: nil,
		},
		{
			name:       "three word command",
			input:      "subs open abc",
			wantLen:    3,
			wantValues: []string{"subs", "open", "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := ParseString(tt.input)
			require.NoError(t, err)
			require.Len(t, c, tt.wantLen)

			if tt.wantValues != nil {
				for i, want := range tt.wantValues {
					assert.Equal(t, Variable(want), c[i].Value)
				}
			}
		})
	}
}
