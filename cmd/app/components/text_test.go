package components

import (
	"testing"
)

func TestHighlightText_NoQuery(t *testing.T) {
	text := "Hello World"
	query := ""

	result := HighlightText(text, query)
	if result == nil {
		t.Error("HighlightText returned nil for empty query")
	}
}

func TestHighlightText_NoText(t *testing.T) {
	text := ""
	query := "test"

	result := HighlightText(text, query)
	if result == nil {
		t.Error("HighlightText returned nil for empty text")
	}
}

func TestHighlightText_SingleMatch(t *testing.T) {
	text := "Hello World"
	query := "World"

	result := HighlightText(text, query)
	if result == nil {
		t.Fatal("HighlightText returned nil")
	}
}

func TestHighlightText_CaseInsensitive(t *testing.T) {
	text := "Hello WORLD"
	query := "world"

	result := HighlightText(text, query)
	if result == nil {
		t.Fatal("HighlightText returned nil")
	}
}

func TestHighlightText_MultipleMatches(t *testing.T) {
	text := "test one test two test three"
	query := "test"

	result := HighlightText(text, query)
	if result == nil {
		t.Fatal("HighlightText returned nil")
	}
}

func TestHighlightText_NoMatch(t *testing.T) {
	text := "Hello World"
	query := "xyz"

	result := HighlightText(text, query)
	if result == nil {
		t.Fatal("HighlightText returned nil")
	}
}

func TestHighlightText_PartialMatch(t *testing.T) {
	text := "Hello World"
	query := "Wor"

	result := HighlightText(text, query)
	if result == nil {
		t.Fatal("HighlightText returned nil")
	}
}

func TestHighlightTextIf_False(t *testing.T) {
	text := "Hello World"
	query := "World"

	result := HighlightTextIf(text, query, false)
	if result == nil {
		t.Fatal("HighlightTextIf returned nil")
	}
}

func TestHighlightTextIf_True(t *testing.T) {
	text := "Hello World"
	query := "World"

	result := HighlightTextIf(text, query, true)
	if result == nil {
		t.Fatal("HighlightTextIf returned nil")
	}
}

func TestHighlightTextIf_EmptyQuery(t *testing.T) {
	text := "Hello World"
	query := ""

	result := HighlightTextIf(text, query, true)
	if result == nil {
		t.Fatal("HighlightTextIf returned nil")
	}
}
