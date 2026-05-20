package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"unicode"
)

type Syntax struct {
	Scanner
}

func NewSyntax(text []rune) *Syntax {
	return &Syntax{Scanner: NewScanner(text)}
}

func (l *Syntax) error() error {
	return fmt.Errorf("syntax error on '%s' line: %d column: %d", string(l.CurrentChar), l.LineNo, l.Column)
}

func (l *Syntax) Parameter() (*Token, error) {
	token := &Token{Type: "", Value: Variable(""), LineNo: l.LineNo, Column: l.Column}

	var result []rune
	if l.CurrentChar == '[' {
		l.Advance()
		result = append(result, l.CurrentChar)

		l.Advance()
		for l.CurrentChar > 0 && l.CurrentChar != ']' {
			result = append(result, l.CurrentChar)
			l.Advance()
		}
		l.Advance()

		token.Type = ParameterToken
		token.Value = Variable(string(result))
	}

	return token, nil
}

func (l *Syntax) GetNextToken() (*Token, error) {
	for l.CurrentChar > 0 {
		if unicode.IsSpace(l.CurrentChar) {
			l.SkipWhitespace()
			continue
		}
		if l.CurrentChar == '[' {
			return l.Parameter()
		}
		if !unicode.IsSpace(l.CurrentChar) {
			return l.Character()
		}

		return nil, l.error()
	}

	return &Token{Type: EOFToken, Value: Variable("")}, nil
}

func collectTokens(define string) ([]*Token, error) {
	s := NewSyntax([]rune(define))
	var tokens []*Token
	token, err := s.GetNextToken()
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, token)
	for token.Type != EOFToken {
		token, err = s.GetNextToken()
		if err != nil {
			return nil, err
		}
		if token.Type != EOFToken {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func validateNumberParam(token *Token) bool {
	n, _ := token.Value.String()
	re := regexp.MustCompile(`\d+`)
	if !re.MatchString(n) {
		return false
	}
	num, err := strconv.ParseInt(n, 10, 64)
	if err == nil {
		token.Value = Variable(num)
	}
	return true
}

func validateBoolParam(token *Token) bool {
	if !(token.Value.Source == "true" || token.Value.Source == "false") {
		return false
	}
	if token.Value.Source == "true" {
		token.Value = Variable(true)
	}
	if token.Value.Source == "false" {
		token.Value = Variable(false)
	}
	return true
}

func SyntaxCheck(define string, actual []*Token) (bool, error) {
	tokens, err := collectTokens(define)
	if err != nil {
		return false, err
	}

	if len(tokens) != len(actual) {
		return false, nil
	}

	res := true
	for i, t := range tokens {
		if t.Type == CharacterToken {
			if t.Value != actual[i].Value {
				res = false
				continue
			}
		}
		if t.Type == ParameterToken {
			switch t.Value.Source {
			case "number":
				if !validateNumberParam(actual[i]) {
					res = false
					continue
				}
			case "bool":
				if !validateBoolParam(actual[i]) {
					res = false
					continue
				}
			case "string":
			case "any":
			}
		}
	}
	return res, nil
}
