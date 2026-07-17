package chatagent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSkillMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		raw         string
		wantName    string
		wantDescSub string
		wantBodySub string
		wantErr     bool
	}{
		{
			name: "valid skill md",
			raw: `---
name: karakeep
description: >-
  Create bookmarks via flowbot bookmark. Use when the user mentions bookmarks.
compatibility: Requires flowbot CLI
---

# Karakeep

Use flowbot bookmark.
`,
			wantName:    "karakeep",
			wantDescSub: "Create bookmarks",
			wantBodySub: "# Karakeep",
		},
		{
			name:    "missing frontmatter",
			raw:     "# No frontmatter\n",
			wantErr: true,
		},
		{
			name: "unterminated frontmatter",
			raw: `---
name: karakeep
description: incomplete
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fm, body, err := parseSkillMarkdown(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, fm.Name)
			require.Contains(t, fm.Description, tt.wantDescSub)
			require.Contains(t, body, tt.wantBodySub)
		})
	}
}
