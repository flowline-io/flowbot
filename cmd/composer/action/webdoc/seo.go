package webdoc

import (
	_ "embed"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
)

//go:embed seo.yaml
var seoYAML []byte

const (
	seoStartMarker = "<!-- seo:start -->"
	seoEndMarker   = "<!-- seo:end -->"
)

type seoConfig struct {
	BaseURL      string          `yaml:"base_url"`
	SitemapPaths []string        `yaml:"sitemap_paths"`
	EntryPages   []seoEntryPage  `yaml:"entry_pages"`
}

type seoEntryPage struct {
	File          string `yaml:"file"`
	Path          string `yaml:"path"`
	Title         string `yaml:"title"`
	Description   string `yaml:"description"`
	WebsiteJSONLD bool   `yaml:"website_json_ld"`
}

func loadSEOConfig() (seoConfig, error) {
	var cfg seoConfig
	if err := yaml.Unmarshal(seoYAML, &cfg); err != nil {
		return seoConfig{}, fmt.Errorf("parsing seo.yaml: %w", err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return seoConfig{}, fmt.Errorf("seo.yaml: base_url is required")
	}
	if len(cfg.SitemapPaths) == 0 {
		return seoConfig{}, fmt.Errorf("seo.yaml: sitemap_paths must not be empty")
	}
	if len(cfg.EntryPages) == 0 {
		return seoConfig{}, fmt.Errorf("seo.yaml: entry_pages must not be empty")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}

func absoluteURL(baseURL, path string) string {
	base := strings.TrimRight(baseURL, "/")
	p := strings.TrimSpace(path)
	if p == "" || p == "/" {
		return base + "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return base + p
}

func buildSitemapXML(baseURL string, paths []string) string {
	var b strings.Builder
	_, _ = b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	_, _ = b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range paths {
		loc := absoluteURL(baseURL, p)
		_, _ = b.WriteString("  <url>\n")
		_, _ = b.WriteString("    <loc>" + loc + "</loc>\n")
		_, _ = b.WriteString("  </url>\n")
	}
	_, _ = b.WriteString("</urlset>\n")
	return b.String()
}

func buildRobotsTxt(baseURL string) string {
	return "User-agent: *\nAllow: /\n\nSitemap: " + absoluteURL(baseURL, "/sitemap.xml") + "\n"
}

func buildEntrySEOBlock(baseURL string, page seoEntryPage) (string, error) {
	canonical := absoluteURL(baseURL, page.Path)
	title := html.EscapeString(page.Title)
	desc := html.EscapeString(page.Description)
	canon := html.EscapeString(canonical)

	var b strings.Builder
	_, _ = b.WriteString("\t\t<title>" + title + "</title>\n")
	_, _ = b.WriteString("\t\t<meta\n")
	_, _ = b.WriteString("\t\t\tname=\"description\"\n")
	_, _ = b.WriteString("\t\t\tcontent=\"" + desc + "\"\n")
	_, _ = b.WriteString("\t\t/>\n")
	_, _ = b.WriteString("\t\t<link rel=\"canonical\" href=\"" + canon + "\" />\n")
	_, _ = b.WriteString("\t\t<meta property=\"og:title\" content=\"" + title + "\" />\n")
	_, _ = b.WriteString("\t\t<meta property=\"og:description\" content=\"" + desc + "\" />\n")
	_, _ = b.WriteString("\t\t<meta property=\"og:url\" content=\"" + canon + "\" />\n")
	_, _ = b.WriteString("\t\t<meta property=\"og:type\" content=\"website\" />\n")
	_, _ = b.WriteString("\t\t<meta property=\"og:site_name\" content=\"Flowbot\" />\n")
	_, _ = b.WriteString("\t\t<meta name=\"twitter:card\" content=\"summary\" />\n")
	_, _ = b.WriteString("\t\t<meta name=\"twitter:title\" content=\"" + title + "\" />\n")
	_, _ = b.WriteString("\t\t<meta name=\"twitter:description\" content=\"" + desc + "\" />\n")
	if page.WebsiteJSONLD {
		ld := struct {
			Context     string `json:"@context"`
			Type        string `json:"@type"`
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
		}{
			Context:     "https://schema.org",
			Type:        "WebSite",
			Name:        "Flowbot",
			URL:         canonical,
			Description: page.Description,
		}
		raw, err := sonic.MarshalIndent(ld, "\t\t\t", "\t")
		if err != nil {
			return "", fmt.Errorf("marshaling WebSite JSON-LD: %w", err)
		}
		_, _ = b.WriteString("\t\t<script type=\"application/ld+json\">\n")
		_, _ = b.WriteString("\t\t\t")
		_, _ = b.Write(raw)
		_, _ = b.WriteString("\n\t\t</script>\n")
	}
	return b.String(), nil
}

func replaceSEOBlock(htmlSrc, block string) (string, error) {
	start := strings.Index(htmlSrc, seoStartMarker)
	end := strings.Index(htmlSrc, seoEndMarker)
	if start < 0 || end < 0 || end < start {
		return "", fmt.Errorf("missing %s / %s markers", seoStartMarker, seoEndMarker)
	}
	var b strings.Builder
	_, _ = b.WriteString(htmlSrc[:start])
	_, _ = b.WriteString(seoStartMarker)
	_, _ = b.WriteString("\n")
	_, _ = b.WriteString(block)
	_, _ = b.WriteString("\t\t")
	_, _ = b.WriteString(seoEndMarker)
	_, _ = b.WriteString(htmlSrc[end+len(seoEndMarker):])
	return b.String(), nil
}

func writeSEOAssets(websiteDir string, cfg seoConfig) error {
	if err := os.WriteFile(
		filepath.Join(websiteDir, "sitemap.xml"),
		[]byte(buildSitemapXML(cfg.BaseURL, cfg.SitemapPaths)),
		0o644,
	); err != nil {
		return fmt.Errorf("writing sitemap.xml: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(websiteDir, "robots.txt"),
		[]byte(buildRobotsTxt(cfg.BaseURL)),
		0o644,
	); err != nil {
		return fmt.Errorf("writing robots.txt: %w", err)
	}
	for _, page := range cfg.EntryPages {
		block, err := buildEntrySEOBlock(cfg.BaseURL, page)
		if err != nil {
			return fmt.Errorf("building SEO for %s: %w", page.File, err)
		}
		path := filepath.Join(websiteDir, page.File)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", page.File, err)
		}
		updated, err := replaceSEOBlock(string(raw), block)
		if err != nil {
			return fmt.Errorf("%s: %w", page.File, err)
		}
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", page.File, err)
		}
	}
	return nil
}
