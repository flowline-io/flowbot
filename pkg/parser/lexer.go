// Package parser provides command and expression parsing.
package parser

import (
	"fmt"
	"unicode"
)

type Token struct {
	Type   string
	Value  Value
	LineNo int
	Column int
}

const (
	CharacterToken = "character"
	ParameterToken = "parameter"
	EOFToken       = "eof"
)

type Lexer struct {
	Scanner
}

func NewLexer(text []rune) *Lexer {
	return &Lexer{Scanner: NewScanner(text)}
}

func (l *Lexer) error() error {
	return fmt.Errorf("lexer error on '%s' line: %d column: %d", string(l.CurrentChar), l.LineNo, l.Column)
}

func (l *Lexer) String() (*Token, error) {
	token := &Token{Type: "", Value: Variable(""), LineNo: l.LineNo, Column: l.Column}

	l.Advance()

	var result []rune
	for l.CurrentChar != '"' {
		result = append(result, l.CurrentChar)
		l.Advance()
	}

	l.Advance()

	s := string(result)
	token.Type = CharacterToken
	token.Value = Variable(s)

	return token, nil
}

func (l *Lexer) GetNextToken() (*Token, error) {
	for l.CurrentChar > 0 {
		if unicode.IsSpace(l.CurrentChar) {
			l.SkipWhitespace()
			continue
		}
		if l.CurrentChar == '"' {
			return l.String()
		}
		if !unicode.IsSpace(l.CurrentChar) {
			return l.Character()
		}
		return nil, l.error()
	}

	return &Token{Type: EOFToken, Value: Variable("")}, nil
}

func ParseString(in string) ([]*Token, error) {
	if in == "" {
		return []*Token{}, nil
	}
	l := NewLexer([]rune(in))
	var tokens []*Token
	token, err := l.GetNextToken()
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, token)
	for token.Type != EOFToken {
		token, err = l.GetNextToken()
		if err != nil {
			return nil, err
		}
		if token.Type != EOFToken {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
