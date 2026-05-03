// Package webdoc generates the Flowbot documentation website from markdown sources.
package webdoc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/russross/blackfriday/v2"
	"github.com/urfave/cli/v3"
)

// Page holds data for the HTML template.
type Page struct {
	Title    string
	Content  template.HTML
	BasePath string
}

// pageTemplate is the HTML wrapper matching the website's visual identity.
const pageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>{{.Title}} — Flowbot</title>
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet" />
<link rel="stylesheet" href="{{.BasePath}}css/style.css" />
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
<li><a href="{{.BasePath}}docs/getting-started.html">Docs</a></li>
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
</div>
<div class="content">
{{.Content}}
</div>
</div>
<footer class="footer">
<div class="footer-left">
<a href="https://github.com/flowline-io/flowbot" target="_blank" rel="noopener">GitHub</a>
<a href="{{.BasePath}}design.html">Design</a>
<a href="{{.BasePath}}api.html">API</a>
<a href="{{.BasePath}}tutorials.html">Tutorials</a>
<a href="{{.BasePath}}docs/getting-started.html">Docs</a>
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
	tpl         = template.Must(template.New("page").Parse(pageTemplate))
	mdLinkRegex = regexp.MustCompile(`href="([^"]+)\.md(#[^"]*)?"`)
)

// WebDocAction generates the documentation website from markdown sources.
func WebDocAction(_ context.Context, _ *cli.Command) error {
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	srcDir := filepath.Join(rootDir, "docs")
	outDir := filepath.Join(srcDir, "website", "docs")

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("cleaning output dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	skipDirs := map[string]bool{
		"api":     true,
		"website": true,
	}

	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
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

		return convertFile(srcDir, outDir, relPath)
	})
	if err != nil {
		return fmt.Errorf("walking docs: %w", err)
	}

	fmt.Println("website docs generated successfully")
	return nil
}

func convertFile(srcDir, outDir, relPath string) error {
	input, err := os.ReadFile(filepath.Join(srcDir, relPath))
	if err != nil {
		return fmt.Errorf("reading %s: %w", relPath, err)
	}

	htmlBody := blackfriday.Run(input,
		blackfriday.WithExtensions(blackfriday.CommonExtensions),
	)

	htmlBody = mdLinkRegex.ReplaceAll(htmlBody, []byte(`href="$1.html$2"`))

	title := extractTitle(input)

	websiteRoot := filepath.Join(srcDir, "website")
	outFile := relPathToOut(relPath)
	absOut := filepath.Join(outDir, outFile)
	outRelDir := filepath.Dir(absOut)
	basePath, err := filepath.Rel(outRelDir, websiteRoot)
	if err != nil {
		basePath = ".."
	}
	basePath = filepath.ToSlash(basePath) + "/"

	if err := os.MkdirAll(filepath.Dir(absOut), 0o755); err != nil {
		return fmt.Errorf("creating dir for %s: %w", outFile, err)
	}

	page := Page{
		Title:    title,
		Content:  template.HTML(htmlBody),
		BasePath: basePath,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, page); err != nil {
		return fmt.Errorf("executing template for %s: %w", relPath, err)
	}

	if err := os.WriteFile(absOut, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}

	fmt.Printf("  %s -> %s\n", relPath, filepath.Join("website", "docs", outFile))
	return nil
}

// relPathToOut maps a source relative path to an output relative path.
func relPathToOut(relPath string) string {
	dir := filepath.Dir(relPath)
	name := filepath.Base(relPath)
	stem := strings.TrimSuffix(name, ".md")

	if stem == "README" && dir != "." {
		return dir + ".html"
	}

	return filepath.Join(dir, stem+".html")
}

// extractTitle returns the first H1 heading text from markdown input.
func extractTitle(input []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return "Documentation"
}
