package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		fallback string
	}{
		{name: "plain title unchanged", input: "Deploy flowbot service", want: "Deploy flowbot service"},
		{name: "strips surrounding quotes", input: `"Fix login bug"`, want: "Fix login bug"},
		{name: "collapses newlines", input: "Plan\nfor\nrelease", want: "Plan for release"},
		{name: "truncates long title", input: "This is an extremely long session title that should be trimmed down for display", want: "This is an extremely long session title that should be trimm"},
		{name: "empty falls back to user text", input: "   ", fallback: "How do I configure Redis?", want: "How do I configure Redis?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTitle(tt.input)
			if got == "" && tt.fallback != "" {
				got = fallbackSessionTitle(tt.fallback)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFallbackSessionTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "short message kept", input: "hello world", want: "hello world"},
		{name: "newline collapsed", input: "line one\nline two", want: "line one line two"},
		{name: "long message truncated", input: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcde..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fallbackSessionTitle(tt.input))
		})
	}
}

func TestBuildSessionTitlePrompt(t *testing.T) {
	tests := []struct {
		name     string
		userText string
		reply    string
		wantSub  []string
	}{
		{name: "includes user and assistant", userText: "hi", reply: "hello", wantSub: []string{"User: hi", "Assistant: hello"}},
		{name: "trims whitespace", userText: "  ask  ", reply: "  ok  ", wantSub: []string{"User: ask", "Assistant: ok"}},
		{name: "truncates long reply", userText: "q", reply: string(make([]byte, 600)), wantSub: []string{"..."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSessionTitlePrompt(tt.userText, tt.reply)
			for _, want := range tt.wantSub {
				assert.Contains(t, got, want)
			}
		})
	}
}
