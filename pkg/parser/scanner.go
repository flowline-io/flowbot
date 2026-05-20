package parser

import "unicode"

// Scanner holds the shared cursor state and common scanning methods for tokenizers.
type Scanner struct {
	Text        []rune
	Pos         int
	CurrentChar rune
	LineNo      int
	Column      int
}

// NewScanner initializes a Scanner with the given input text.
func NewScanner(text []rune) Scanner {
	return Scanner{Text: text, Pos: 0, CurrentChar: text[0], LineNo: 1, Column: 1}
}

// Advance moves the cursor forward by one rune, tracking line and column numbers.
func (s *Scanner) Advance() {
	if s.CurrentChar == '\n' {
		s.LineNo++
		s.Column = 0
	}
	s.Pos++
	if s.Pos > len(s.Text)-1 {
		s.CurrentChar = 0
	} else {
		s.CurrentChar = s.Text[s.Pos]
		s.Column++
	}
}

// Peek returns the next rune without advancing the cursor.
func (s *Scanner) Peek() rune {
	peekPos := s.Pos + 1
	if peekPos > len(s.Text)-1 {
		return 0
	}
	return s.Text[peekPos]
}

// SkipWhitespace advances past all whitespace characters.
func (s *Scanner) SkipWhitespace() {
	for s.CurrentChar > 0 && unicode.IsSpace(s.CurrentChar) {
		s.Advance()
	}
}

// Character consumes a run of non-space characters into a Token.
func (s *Scanner) Character() (*Token, error) {
	token := &Token{Type: "", Value: Variable(""), LineNo: s.LineNo, Column: s.Column}

	var result []rune
	for s.CurrentChar > 0 && !unicode.IsSpace(s.CurrentChar) {
		result = append(result, s.CurrentChar)
		s.Advance()
	}

	val := string(result)
	token.Type = CharacterToken
	token.Value = Variable(val)

	return token, nil
}
