package webdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantFM FrontMatter
		wantMD string
	}{
		{
			name:   "no front matter",
			input:  "# Title\n\nContent here.",
			wantFM: FrontMatter{},
			wantMD: "# Title\n\nContent here.",
		},
		{
			name: "valid front matter all fields",
			input: `---
title: Custom Title
description: A helpful description
accent_color: "#ff6b35"
wide: true
hide_sidebar: true
---
# Body heading

Some content.`,
			wantFM: FrontMatter{
				Title:       "Custom Title",
				Description: "A helpful description",
				AccentColor: "#ff6b35",
				Wide:        true,
				HideSidebar: true,
			},
			wantMD: "# Body heading\n\nSome content.",
		},
		{
			name: "front matter partial fields",
			input: `---
title: My Page
---
Start of content.`,
			wantFM: FrontMatter{
				Title: "My Page",
			},
			wantMD: "Start of content.",
		},
		{
			name:   "empty front matter",
			input:  "---\n---\nBody starts here.",
			wantFM: FrontMatter{},
			wantMD: "---\n---\nBody starts here.",
		},
		{
			name:   "invalid yaml treated as no front matter",
			input:  "---\n\tbad: [[[yaml\n---\n# Heading\nText.",
			wantFM: FrontMatter{},
			wantMD: "---\n\tbad: [[[yaml\n---\n# Heading\nText.",
		},
		{
			name:   "only opening delimiter no closing",
			input:  "---\ntitle: Test\n\n# Content.",
			wantFM: FrontMatter{},
			wantMD: "---\ntitle: Test\n\n# Content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFM, gotMD := parseFrontMatter([]byte(tt.input))
			assert.Equal(t, tt.wantFM, gotFM)
			assert.Equal(t, tt.wantMD, string(gotMD))
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		fm    FrontMatter
		want  string
	}{
		{
			name:  "front matter title wins",
			input: "# H1 Title",
			fm:    FrontMatter{Title: "FM Title"},
			want:  "FM Title",
		},
		{
			name:  "h1 fallback when no fm title",
			input: "# Main Heading",
			fm:    FrontMatter{Description: "desc"},
			want:  "Main Heading",
		},
		{
			name:  "documentation fallback",
			input: "Just some text, no heading.",
			fm:    FrontMatter{},
			want:  "Documentation",
		},
		{
			name:  "h1 requires space after #",
			input: "## Not H1\n#Actual H1 should be `# `",
			fm:    FrontMatter{},
			want:  "Documentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle([]byte(tt.input), tt.fm)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRelPathToOut(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		want    string
	}{
		{
			name:    "README becomes index.html in same dir",
			relPath: "getting-started/README.md",
			want:    "getting-started/index.html",
		},
		{
			name:    "regular markdown becomes html",
			relPath: "user-guide/pipeline.md",
			want:    "user-guide/pipeline.html",
		},
		{
			name:    "root README",
			relPath: "README.md",
			want:    "index.html",
		},
		{
			name:    "nested regular file",
			relPath: "a/b/c/doc.md",
			want:    "a/b/c/doc.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relPathToOut(tt.relPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOutURL(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		want    string
	}{
		{
			name:    "README index page",
			relPath: "getting-started/README.md",
			want:    "docs/getting-started/",
		},
		{
			name:    "regular page",
			relPath: "user-guide/pipeline.md",
			want:    "docs/user-guide/pipeline.html",
		},
		{
			name:    "nested README",
			relPath: "developer-guide/sub/README.md",
			want:    "docs/developer-guide/sub/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outURL(tt.relPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDirToTitle(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "kebab case", dir: "getting-started", want: "Getting Started"},
		{name: "single word", dir: "architecture", want: "Architecture"},
		{name: "multi word", dir: "user-guide", want: "User Guide"},
		{name: "three words", dir: "developer-guide", want: "Developer Guide"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dirToTitle(tt.dir)
			assert.Equal(t, tt.want, got)
		})
	}
}
