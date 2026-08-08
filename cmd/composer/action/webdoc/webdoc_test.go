package webdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontMatter(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			gotFM, gotMD := parseFrontMatter([]byte(tt.input))
			assert.Equal(t, tt.wantFM, gotFM)
			assert.Equal(t, tt.wantMD, string(gotMD))
		})
	}
}

func TestExtractTitle(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := extractTitle([]byte(tt.input), tt.fm)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRelPathToOut(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := relPathToOut(tt.relPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOutURL(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := outURL(tt.relPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDirToTitle(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := dirToTitle(tt.dir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAbsoluteURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{name: "home slash", base: "https://flowline-io.github.io", path: "/", want: "https://flowline-io.github.io/"},
		{name: "home empty", base: "https://flowline-io.github.io/", path: "", want: "https://flowline-io.github.io/"},
		{name: "html page", base: "https://flowline-io.github.io", path: "/design.html", want: "https://flowline-io.github.io/design.html"},
		{name: "docs path no leading slash", base: "https://flowline-io.github.io", path: "docs/getting-started/", want: "https://flowline-io.github.io/docs/getting-started/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, absoluteURL(tt.base, tt.path))
		})
	}
}

func TestLoadSEOConfig(t *testing.T) {
	t.Parallel()
	cfg, err := loadSEOConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://flowline-io.github.io", cfg.BaseURL)
	assert.Equal(t, []string{
		"/",
		"/design.html",
		"/api.html",
		"/tutorials.html",
		"/skills.html",
	}, cfg.SitemapPaths)
	require.NotEmpty(t, cfg.EntryPages)
	assert.Equal(t, "index.html", cfg.EntryPages[0].File)
	assert.True(t, cfg.EntryPages[0].WebsiteJSONLD)
}

func TestBuildSitemapXML(t *testing.T) {
	t.Parallel()
	got := buildSitemapXML("https://flowline-io.github.io", []string{"/", "/design.html"})
	assert.Contains(t, got, `<loc>https://flowline-io.github.io/</loc>`)
	assert.Contains(t, got, `<loc>https://flowline-io.github.io/design.html</loc>`)
	assert.Contains(t, got, `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`)
}

func TestBuildRobotsTxt(t *testing.T) {
	t.Parallel()
	got := buildRobotsTxt("https://flowline-io.github.io")
	assert.Contains(t, got, "User-agent: *")
	assert.Contains(t, got, "Allow: /")
	assert.Contains(t, got, "Sitemap: https://flowline-io.github.io/sitemap.xml")
}

func TestBuildEntrySEOBlock(t *testing.T) {
	t.Parallel()
	block, err := buildEntrySEOBlock("https://flowline-io.github.io", seoEntryPage{
		Path:          "/design.html",
		Title:         "Architecture & Design — Flowbot",
		Description:   "Design overview.",
		WebsiteJSONLD: false,
	})
	require.NoError(t, err)
	assert.Contains(t, block, "<title>Architecture &amp; Design — Flowbot</title>")
	assert.Contains(t, block, `rel="canonical" href="https://flowline-io.github.io/design.html"`)
	assert.Contains(t, block, `property="og:title"`)
	assert.Contains(t, block, `name="twitter:card" content="summary"`)
	assert.NotContains(t, block, "application/ld+json")
}

func TestBuildEntrySEOBlockJSONLD(t *testing.T) {
	t.Parallel()
	block, err := buildEntrySEOBlock("https://flowline-io.github.io", seoEntryPage{
		Path:          "/",
		Title:         "Flowbot",
		Description:   "Homelab orchestration.",
		WebsiteJSONLD: true,
	})
	require.NoError(t, err)
	assert.Contains(t, block, `"@type": "WebSite"`)
	assert.Contains(t, block, `"url": "https://flowline-io.github.io/"`)
}

func TestReplaceSEOBlock(t *testing.T) {
	t.Parallel()
	src := "<head>\n\t\t<!-- seo:start -->\n\t\told\n\t\t<!-- seo:end -->\n\t\t<link />\n</head>"
	got, err := replaceSEOBlock(src, "\t\t<title>New</title>\n")
	require.NoError(t, err)
	assert.Contains(t, got, "<!-- seo:start -->\n\t\t<title>New</title>\n\t\t<!-- seo:end -->")
	assert.NotContains(t, got, "old")
	assert.Contains(t, got, "<link />")
}
