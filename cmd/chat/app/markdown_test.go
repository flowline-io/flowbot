package app

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderDebounce(t *testing.T) {
	tests := []struct {
		name string
		buf  string
		want time.Duration
	}{
		{name: "short text", buf: "hello", want: debounceShort},
		{name: "medium text", buf: string(make([]byte, 2000)), want: debounceMid},
		{name: "long text", buf: string(make([]byte, 6000)), want: debounceLong},
		{name: "code fence at newline", buf: "text\n```go\nline\n", want: debounceCodeLine},
		{name: "code fence mid line", buf: "text\n```go\nline", want: debounceCodeMidLine},
		{name: "long code fence mid line", buf: string(make([]byte, 6000)) + "```go\nx", want: debounceLong + 200*time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderDebounce(tt.buf)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "plain text", src: "hello world"},
		{name: "code fence", src: "```go\nfmt.Println(\"hi\")\n```"},
		{name: "header line", src: "# Title\nbody"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := RenderMarkdown(tt.src, 80)
			assert.NotEmpty(t, out)
		})
	}
}

func TestRenderMarkdownNoDuplicatePlainText(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "plain sentence", src: "Hello! How can I help?", want: "Hello! How can I help?"},
		{name: "multiline body", src: "line one\nline two", want: "line one"},
		{name: "text before code fence", src: "intro\n```go\nx := 1\n```", want: "intro"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := stripANSI(RenderMarkdown(tt.src, 80))
			if tt.name == "text before code fence" {
				assert.Contains(t, out, tt.want)
				assert.Equal(t, 1, strings.Count(out, tt.want))
				return
			}
			assert.Contains(t, out, tt.want)
			assert.Equal(t, 1, strings.Count(out, tt.want))
			if tt.name == "multiline body" {
				assert.Contains(t, out, "line two")
				assert.Equal(t, 1, strings.Count(out, "line two"))
			}
		})
	}
}
