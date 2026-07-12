package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatAgentMarkdownSanitizerPreservesKatex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		text       string
		wantSubstr []string
		wantAbsent []string
	}{
		{
			name:       "table inline math",
			text:       "| A | B |\n| --- | --- |\n| $10^0 = 1$ | $\\lg 1 = 0$ |",
			wantSubstr: []string{"class=\"katex\"", "class=\"katex-html\"", "10", "lg"},
			wantAbsent: []string{"$10^0 = 1$", "javascript:"},
		},
		{
			name:       "block math",
			text:       "$$\nA = \\pi r^2\n$$\n",
			wantSubstr: []string{"class=\"katex-display\"", "katex"},
			wantAbsent: []string{"$$"},
		},
		{
			name:       "strips script tags",
			text:       "<script>alert(1)</script>\n\n$x^2$\n",
			wantSubstr: []string{"katex"},
			wantAbsent: []string{"<script>", "alert(1)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RenderChatAgentMarkdownHTML(tt.text)
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, got, sub)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, got, absent)
			}
		})
	}
}
