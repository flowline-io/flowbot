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
