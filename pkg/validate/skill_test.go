package validate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestSkillName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		skill   string
		wantErr string
	}{
		{name: "valid simple", skill: "github"},
		{name: "valid hyphenated", skill: "pdf-processing"},
		{name: "valid digits", skill: "skill2"},
		{name: "empty", skill: "", wantErr: "non-empty"},
		{name: "uppercase", skill: "PDF-Processing", wantErr: "lowercase"},
		{name: "leading hyphen", skill: "-pdf", wantErr: "start or end"},
		{name: "trailing hyphen", skill: "pdf-", wantErr: "start or end"},
		{name: "consecutive hyphens", skill: "pdf--processing", wantErr: "consecutive"},
		{name: "underscore", skill: "pdf_processing", wantErr: "invalid characters"},
		{name: "space", skill: "pdf processing", wantErr: "invalid characters"},
		{name: "too long", skill: strings.Repeat("a", validate.SkillNameMaxLen+1), wantErr: "exceeds"},
		{name: "max length ok", skill: strings.Repeat("a", validate.SkillNameMaxLen)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validate.SkillName(tt.skill)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSkillDescription(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		desc    string
		wantErr string
	}{
		{name: "valid", desc: "Does a thing. Use when needed."},
		{name: "empty", desc: "  ", wantErr: "non-empty"},
		{name: "too long", desc: strings.Repeat("x", validate.SkillDescMaxLen+1), wantErr: "exceeds"},
		{name: "max length ok", desc: strings.Repeat("x", validate.SkillDescMaxLen)},
		{name: "unicode runes counted", desc: strings.Repeat("é", validate.SkillDescMaxLen+1), wantErr: "exceeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validate.SkillDescription(tt.desc)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSkillCompatibility(t *testing.T) {
	t.Parallel()
	require.NoError(t, validate.SkillCompatibility(""))
	require.NoError(t, validate.SkillCompatibility("Requires flowbot CLI"))
	err := validate.SkillCompatibility(strings.Repeat("c", validate.SkillCompatibilityMaxLen+1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds")
}

func TestSkillNameMatchesDir(t *testing.T) {
	t.Parallel()
	require.NoError(t, validate.SkillNameMatchesDir("github", "github"))
	require.NoError(t, validate.SkillNameMatchesDir("github", "."))
	require.NoError(t, validate.SkillNameMatchesDir("github", ""))
	err := validate.SkillNameMatchesDir("github", "gitlab")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must match")
}

func TestSkillDocument(t *testing.T) {
	t.Parallel()
	require.NoError(t, validate.SkillDocument("github", "Inspect GitHub via flowbot.", "Requires CLI", "github"))
	require.NoError(t, validate.SkillDocument("github", "Inspect GitHub via flowbot.", "", "github"))
	require.Error(t, validate.SkillDocument("github", "desc", strings.Repeat("c", validate.SkillCompatibilityMaxLen+1), "github"))
}

func TestParseSkillMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		raw         string
		wantName    string
		wantDescSub string
		wantCompat  string
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
			wantCompat:  "Requires flowbot CLI",
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
			fm, body, err := validate.ParseSkillMarkdown(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, fm.Name)
			require.Contains(t, fm.Description, tt.wantDescSub)
			require.Equal(t, tt.wantCompat, strings.TrimSpace(fm.Compatibility))
			require.Contains(t, body, tt.wantBodySub)
		})
	}
}
