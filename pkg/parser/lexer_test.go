package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	l := NewLexer([]rune("subs  open abc \"a b c\""))
	token, err := l.GetNextToken()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, CharacterToken, token.Type)
	token, err = l.GetNextToken()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, CharacterToken, token.Type)
	token, err = l.GetNextToken()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, CharacterToken, token.Type)
	token, err = l.GetNextToken()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, CharacterToken, token.Type)
	token, err = l.GetNextToken()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, EOFToken, token.Type)
}

func TestParseCommand(t *testing.T) {
	c, err := ParseString("subs")
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, c, 1)

	c, err = ParseString("subs list")
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, c, 2)

	c, err = ParseString("subs open abc")
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, c, 3)

	require.Equal(t, Variable("subs"), c[0].Value)
	require.Equal(t, Variable("open"), c[1].Value)
	require.Equal(t, Variable("abc"), c[2].Value)
}
