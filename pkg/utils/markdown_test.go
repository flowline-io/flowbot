package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSubstr []string
		wantEmpty  bool
	}{
		{
			name:       "heading",
			source:     "# Title\n",
			wantSubstr: []string{"<h1", "Title", "</h1>"},
		},
		{
			name:       "fenced code block",
			source:     "```go\nfmt.Println(\"hi\")\n```\n",
			wantSubstr: []string{"<pre><code", "fmt.Println"},
		},
		{
			name: "gfm table",
			source: `| Name | Value |
| --- | --- |
| foo | bar |
`,
			wantSubstr: []string{"<table>", "<th", "Name", "foo"},
		},
		{
			name:       "autolink",
			source:     "Visit https://example.com now.\n",
			wantSubstr: []string{`<a href="https://example.com"`},
		},
		{
			name:       "strikethrough",
			source:     "~~removed~~\n",
			wantSubstr: []string{"<del>removed</del>"},
		},
		{
			name:       "inline math",
			source:     "Exponent: $10^2 = 100$\n",
			wantSubstr: []string{"katex", "katex-html"},
		},
		{
			name:      "empty input",
			source:    "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MarkdownToHTML([]byte(tt.source))
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, strings.TrimSpace(string(got)))
				return
			}

			html := string(got)
			for _, substr := range tt.wantSubstr {
				assert.Contains(t, html, substr)
			}
		})
	}
}

func TestMarkdownToSafeHTML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		source     string
		wantSubstr []string
		wantAbsent []string
	}{
		{
			name:       "strips script tags from embedded HTML",
			source:     "<script>alert(1)</script>\n\nHello\n",
			wantSubstr: []string{"Hello"},
			wantAbsent: []string{"<script>", "alert(1)"},
		},
		{
			name:       "strips onclick handlers",
			source:     `<a href="https://example.com" onclick="alert(1)">link</a>`,
			wantSubstr: []string{`href="https://example.com"`, "link"},
			wantAbsent: []string{"onclick", "alert(1)"},
		},
		{
			name:       "preserves katex classes",
			source:     "Exponent: $10^2 = 100$\n",
			wantSubstr: []string{"katex", "katex-html"},
			wantAbsent: []string{"<script>"},
		},
		{
			name:       "preserves gfm table",
			source:     "| A | B |\n| --- | --- |\n| 1 | 2 |\n",
			wantSubstr: []string{"<table>", "<th", "A", "1"},
			wantAbsent: []string{"<script>"},
		},
		{
			name:       "javascript href stripped",
			source:     `[x](javascript:alert(1))`,
			wantSubstr: []string{"x"},
			wantAbsent: []string{"javascript:", "alert(1)"},
		},
		{
			name:       "external markdown link opens in new tab",
			source:     "[docs](https://example.com/path)\n",
			wantSubstr: []string{`href="https://example.com/path"`, `target="_blank"`, `noopener`},
			wantAbsent: []string{"javascript:"},
		},
		{
			name:       "autolink opens in new tab",
			source:     "Visit https://example.com now.\n",
			wantSubstr: []string{`href="https://example.com"`, `target="_blank"`, `noopener`},
			wantAbsent: []string{"javascript:"},
		},
		{
			name:       "relative link stays same tab",
			source:     "[local](/service/web/agents)\n",
			wantSubstr: []string{`href="/service/web/agents"`},
			wantAbsent: []string{`target="_blank"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MarkdownToSafeHTML([]byte(tt.source))
			require.NoError(t, err)
			html := string(got)
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, html, sub)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, html, absent)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantSubstr []string
		wantAbsent []string
	}{
		{
			name:       "strips script",
			input:      `<p>ok</p><script>evil()</script>`,
			wantSubstr: []string{"ok"},
			wantAbsent: []string{"<script>", "evil()"},
		},
		{
			name:       "keeps safe paragraph",
			input:      `<p class="note">safe</p>`,
			wantSubstr: []string{"safe"},
			wantAbsent: []string{"<script>"},
		},
		{
			name:       "empty input",
			input:      "",
			wantSubstr: nil,
			wantAbsent: []string{"<script>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(SanitizeHTML([]byte(tt.input)))
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, got, sub)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, got, absent)
			}
		})
	}
}
