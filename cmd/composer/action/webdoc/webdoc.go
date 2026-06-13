// Package webdoc generates the Flowbot documentation website from markdown sources.
package webdoc

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/microcosm-cc/bluemonday"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/pkg/utils"
)

// FrontMatter holds the parsed YAML front matter from a markdown file.
// Fields are optional; zero values mean "not set" and will not override defaults.
type FrontMatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	AccentColor string `yaml:"accent_color"`
	Wide        bool   `yaml:"wide"`
	HideSidebar bool   `yaml:"hide_sidebar"`
}

// DocSection represents a top-level documentation section with its pages.
type DocSection struct {
	Title string
	Pages []DocNavPage
}

// DocNavPage represents a single page in the docs sidebar navigation.
type DocNavPage struct {
	Title  string
	URL    string // relative to docs/website/, e.g. "docs/getting-started/"
	Active bool
	IsSub  bool
}

// Page holds data for the HTML template.
type Page struct {
	Title       string
	Description string
	Content     template.HTML
	BasePath    string
	DocSections []DocSection
	AccentColor string
	Wide        bool
	HideSidebar bool
}

// pageTemplate is the HTML wrapper matching the website's visual identity.
const pageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>{{.Title}} — Flowbot</title>
{{if .Description}}<meta name="description" content="{{.Description}}" />{{end}}
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet" />
<link rel="stylesheet" href="{{.BasePath}}css/style.css" />
{{if .AccentColor}}<style>:root { --page-accent: {{.AccentColor}}; --page-accent-dim: color-mix(in srgb, {{.AccentColor}} 12%, transparent); --page-accent-glow: color-mix(in srgb, {{.AccentColor}} 25%, transparent); } .docs-content h3 { color: var(--page-accent); } .docs-content a { color: var(--page-accent); } .docs-content a:hover { color: var(--green); } .docs-content pre::before { background: linear-gradient(90deg, transparent, var(--page-accent), transparent); } .docs-content blockquote { border-left-color: var(--page-accent); background: var(--page-accent-dim); } .page-hero h1 em { background: linear-gradient(135deg, var(--page-accent), var(--green)); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }</style>{{end}}
<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><circle cx='50' cy='50' r='40' fill='none' stroke='%2300e5ff' stroke-width='3'/><circle cx='50' cy='50' r='15' fill='%2300e5ff'/></svg>" />
</head>
<body>
<div class="bg-grid"><canvas id="bg-canvas"></canvas></div>
<nav class="nav">
<div class="nav-inner">
<a href="{{.BasePath}}index.html" class="nav-logo">flowbot<span>.io</span></a>
<ul class="nav-links">
<li><a href="{{.BasePath}}index.html#overview">Overview</a></li>
<li><a href="{{.BasePath}}index.html#problem">Problem</a></li>
<li><a href="{{.BasePath}}index.html#capabilities">Capabilities</a></li>
<li><a href="{{.BasePath}}index.html#workflow">Workflow</a></li>
<li><a href="{{.BasePath}}design.html">Design</a></li>
<li><a href="{{.BasePath}}api.html">API</a></li>
<li><a href="{{.BasePath}}tutorials.html">Tutorials</a></li>
<li><a href="{{.BasePath}}docs/getting-started/">Docs</a></li>
</ul>
<a href="https://github.com/flowline-io/flowbot" class="nav-cta" target="_blank" rel="noopener">GitHub</a>
<button class="nav-toggle" aria-label="Menu">
<span></span><span></span><span></span>
</button>
</div>
</nav>
<div class="page">
<div class="page-hero">
<h1>{{.Title}}</h1>
{{if .Description}}<p class="docs-hero-desc">{{.Description}}</p>{{end}}
</div>
<div class="docs-layout{{if .Wide}} docs-layout-wide{{end}}{{if .HideSidebar}} docs-layout-no-sidebar{{end}}">
{{if not .HideSidebar}}
<aside class="docs-sidebar">
<nav class="docs-nav">
{{range .DocSections}}
<div class="docs-nav-section">
<h3 class="docs-nav-title">{{.Title}}</h3>
<ul class="docs-nav-items">
			{{range .Pages}}
			<li><a href="{{$.BasePath}}{{.URL}}" class="docs-nav-link{{if .IsSub}} docs-nav-sub{{end}}{{if .Active}} active{{end}}">{{.Title}}</a></li>
			{{end}}
</ul>
</div>
{{end}}
</nav>
</aside>
{{end}}
<main class="docs-content">
{{.Content}}
</main>
</div>
</div>
<footer class="footer">
<div class="footer-left">
<a href="https://github.com/flowline-io/flowbot" target="_blank" rel="noopener">GitHub</a>
<a href="{{.BasePath}}design.html">Design</a>
<a href="{{.BasePath}}api.html">API</a>
<a href="{{.BasePath}}tutorials.html">Tutorials</a>
<a href="{{.BasePath}}docs/getting-started/">Docs</a>
<span style="color: var(--white-muted); font-size: 0.82rem">GPL-3.0</span>
</div>
<div class="footer-right">
Built for the Homelabbers, by the Homelabbers.
</div>
</footer>
<script src="{{.BasePath}}js/main.js"></script>
</body>
</html>`

var (
	tpl             = template.Must(template.New("page").Parse(pageTemplate))
	mdLinkRegex     = regexp.MustCompile(`href="([^"]+)\.md(#[^"]*)?"`)
	readmeLinkRegex = regexp.MustCompile(`href="([^"]*/)README\.html(#[^"]*)?"`)
)

// docPageInfo holds metadata about a single documentation page.
type docPageInfo struct {
	SourcePath  string // e.g. "getting-started/README.md"
	Title       string
	OutURL      string // e.g. "docs/getting-started/" or "docs/user-guide/pipeline.html"
	FrontMatter FrontMatter
}

// WebDocAction generates the documentation website from markdown sources.
func WebDocAction(_ *cobra.Command, _ []string) error {
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	srcDir := filepath.Join(rootDir, "docs")
	outDir := filepath.Join(srcDir, "website", "docs")

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("cleaning output dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	var allPages []docPageInfo
	if err := collectPages(srcDir, &allPages); err != nil {
		return fmt.Errorf("collecting pages: %w", err)
	}

	websiteRoot := filepath.Join(srcDir, "website")
	for i := range allPages {
		if err := convertFile(srcDir, outDir, &allPages[i], i, allPages, websiteRoot); err != nil {
			return err
		}
	}

	_, _ = fmt.Println("website docs generated successfully")
	return nil
}

// collectPages walks the docs source directory and collects page metadata.
func collectPages(srcDir string, pages *[]docPageInfo) error {
	skipDirs := map[string]bool{
		"api":         true,
		"website":     true,
		"superpowers": true,
	}

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) > 0 && skipDirs[parts[0]] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if relPath == "README.md" {
			return nil
		}

		input, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", relPath, err)
		}

		fm, content := parseFrontMatter(input)
		title := extractTitle(content, fm)
		*pages = append(*pages, docPageInfo{
			SourcePath:  relPath,
			Title:       title,
			OutURL:      outURL(relPath),
			FrontMatter: fm,
		})
		return nil
	})
}

// buildSectionsWithActive builds the navigation section list with the
// current page (at activeIndex) marked as active. Sections are discovered
// automatically from the source directory structure.
func buildSectionsWithActive(pages []docPageInfo, activeIndex int) []DocSection {
	secMap := make(map[string][]DocNavPage)
	for i, p := range pages {
		dir := strings.SplitN(p.SourcePath, "/", 2)[0]
		secMap[dir] = append(secMap[dir], DocNavPage{
			Title:  p.Title,
			URL:    p.OutURL,
			Active: i == activeIndex,
		})
	}

	dirs := sortedSectionDirs(secMap)

	var sections []DocSection
	for _, dir := range dirs {
		items := secMap[dir]
		hasIndex := false
		for _, item := range items {
			if strings.HasSuffix(item.URL, "/") {
				hasIndex = true
				break
			}
		}
		// Sort: README/index pages first, then alphabetical.
		slices.SortStableFunc(items, func(a, b DocNavPage) int {
			aIdx := strings.HasSuffix(a.URL, "/")
			bIdx := strings.HasSuffix(b.URL, "/")
			if aIdx != bIdx {
				if aIdx {
					return -1
				}
				return 1
			}
			if a.Title < b.Title {
				return -1
			}
			if a.Title > b.Title {
				return 1
			}
			return 0
		})

		if hasIndex {
			for i := range items {
				if !strings.HasSuffix(items[i].URL, "/") {
					items[i].IsSub = true
				}
			}
		}

		secTitle := sectionTitle(items, dir)
		sections = append(sections, DocSection{
			Title: secTitle,
			Pages: items,
		})
	}
	return sections
}

// sortedSectionDirs returns section directory names in a stable order.
// "getting-started" (if present) is placed first, then the rest alphabetically.
func sortedSectionDirs(secMap map[string][]DocNavPage) []string {
	dirs := make([]string, 0, len(secMap))
	for d := range secMap {
		dirs = append(dirs, d)
	}
	slices.Sort(dirs)

	// Push "getting-started" to the front.
	for i, d := range dirs {
		if d == "getting-started" {
			copy(dirs[1:i+1], dirs[0:i])
			dirs[0] = "getting-started"
			break
		}
	}
	return dirs
}

// sectionTitle returns the display name for a section. Uses the README/index
// page title if available, otherwise humanizes the directory name.
func sectionTitle(items []DocNavPage, dir string) string {
	for _, item := range items {
		if strings.HasSuffix(item.URL, "/") {
			return item.Title
		}
	}
	return dirToTitle(dir)
}

// dirToTitle converts a kebab-case directory name to a human-readable title.
func dirToTitle(dir string) string {
	words := strings.Split(dir, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// outURL maps a source relative path to a web URL relative to docs/website/.
func outURL(relPath string) string {
	outFile := relPathToOut(relPath)
	url := "docs/" + filepath.ToSlash(outFile)
	if strings.HasSuffix(url, "/index.html") {
		url = url[:len(url)-len("index.html")]
	}
	return url
}

func convertFile(srcDir, outDir string, info *docPageInfo, activeIndex int, allPages []docPageInfo, websiteRoot string) error {
	input, err := os.ReadFile(filepath.Join(srcDir, info.SourcePath))
	if err != nil {
		return fmt.Errorf("reading %s: %w", info.SourcePath, err)
	}

	_, content := parseFrontMatter(input)

	htmlBody, err := utils.MarkdownToHTML(content)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", info.SourcePath, err)
	}
	htmlBody = bluemonday.UGCPolicy().SanitizeBytes(htmlBody)

	htmlBody = mdLinkRegex.ReplaceAll(htmlBody, []byte(`href="$1.html$2"`))
	htmlBody = readmeLinkRegex.ReplaceAll(htmlBody, []byte(`href="$1$2"`))

	safeContent := template.HTML(htmlBody) // #nosec G203 -- content sanitized by bluemonday above

	outFile := relPathToOut(info.SourcePath)
	absOut := filepath.Join(outDir, outFile)
	outRelDir := filepath.Dir(absOut)
	basePath, err := filepath.Rel(outRelDir, websiteRoot)
	if err != nil {
		basePath = ".."
	}
	basePath = filepath.ToSlash(basePath) + "/"

	if err := os.MkdirAll(filepath.Dir(absOut), 0o750); err != nil {
		return fmt.Errorf("creating dir for %s: %w", outFile, err)
	}

	page := Page{
		Title:       info.Title,
		Description: info.FrontMatter.Description,
		Content:     safeContent,
		BasePath:    basePath,
		DocSections: buildSectionsWithActive(allPages, activeIndex),
		AccentColor: info.FrontMatter.AccentColor,
		Wide:        info.FrontMatter.Wide,
		HideSidebar: info.FrontMatter.HideSidebar,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, page); err != nil {
		return fmt.Errorf("executing template for %s: %w", info.SourcePath, err)
	}

	if err := os.WriteFile(absOut, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}

	_, _ = fmt.Printf("  %s -> %s\n", info.SourcePath, filepath.Join("website", "docs", outFile))
	return nil
}

// relPathToOut maps a source relative path to an output relative path.
// README.md files become index.html preserving their directory context so
// that relative links between pages resolve correctly on GitHub Pages.
// Always uses forward slashes since the result is used in web URLs.
func relPathToOut(relPath string) string {
	dir := filepath.Dir(relPath)
	name := filepath.Base(relPath)
	stem := strings.TrimSuffix(name, ".md")

	if stem == "README" {
		return filepath.ToSlash(filepath.Join(dir, "index.html"))
	}

	return filepath.ToSlash(filepath.Join(dir, stem+".html"))
}

// parseFrontMatter extracts YAML front matter delimited by --- and returns the
// parsed metadata together with the remaining markdown content. If no front
// matter is present, fm remains the zero value and content equals input.
func parseFrontMatter(input []byte) (fm FrontMatter, content []byte) {
	const delimiter = "---"
	scanner := bufio.NewScanner(bytes.NewReader(input))

	if !scanner.Scan() || scanner.Text() != delimiter {
		return fm, input
	}

	var yamlLines []string
	foundClosing := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == delimiter {
			foundClosing = true
			break
		}
		yamlLines = append(yamlLines, line)
	}

	if !foundClosing || len(yamlLines) == 0 {
		return fm, input
	}

	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return FrontMatter{}, input
	}

	rest := make([]byte, 0, len(input))
	if scanner.Scan() {
		rest = append(rest, scanner.Bytes()...)
		rest = append(rest, '\n')
	}
	for scanner.Scan() {
		rest = append(rest, scanner.Bytes()...)
		rest = append(rest, '\n')
	}

	return fm, bytes.TrimRight(rest, "\n")
}

// extractTitle returns the best available title for a documentation page.
// It checks front matter first, then falls back to the first H1 heading,
// and finally returns "Documentation" as a last resort.
func extractTitle(input []byte, fm FrontMatter) string {
	if fm.Title != "" {
		return fm.Title
	}
	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return "Documentation"
}
