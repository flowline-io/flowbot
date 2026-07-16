package utils

import (
	"bytes"
	"regexp"
	"sync"

	katex "github.com/FurqanSoftware/goldmark-katex"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	markdownSanitizeOnce sync.Once
	markdownSanitize     *bluemonday.Policy
)

// MarkdownToHTML converts GitHub-flavored markdown into HTML.
// goldmark uses html.WithUnsafe so embedded HTML in markdown is preserved;
// callers that render in a browser MUST sanitize (prefer MarkdownToSafeHTML).
func MarkdownToHTML(source []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.DefinitionList,
			extension.TaskList,
			&katex.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			// Required so markdown-embedded HTML (and KaTeX output paths) survive conversion.
			// Always pair with SanitizeHTML / MarkdownToSafeHTML before templ.Raw or template.HTML.
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarkdownSanitizePolicy returns bluemonday.UGCPolicy extended for KaTeX MathML
// and layout attributes produced by goldmark-katex.
func MarkdownSanitizePolicy() *bluemonday.Policy {
	markdownSanitizeOnce.Do(func() {
		p := bluemonday.UGCPolicy()
		p.AllowElements(
			"math", "semantics", "mrow", "msup", "msub", "msubsup", "mfrac", "msqrt", "mroot",
			"mi", "mn", "mo", "mtext", "mspace", "mstyle", "annotation", "mpadded", "mphantom",
			"menclose", "mover", "munder", "munderover", "mtable", "mtr", "mtd", "maligngroup",
			"malignmark", "mlabeledtr", "merror", "mprescripts", "none",
		)
		p.AllowAttrs("xmlns").Matching(regexp.MustCompile(`^http://www\.w3\.org/1998/Math/MathML$`)).OnElements("math")
		p.AllowAttrs("encoding").Matching(regexp.MustCompile(`^application/x-tex$`)).OnElements("annotation")
		p.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)).OnElements("span")
		p.AllowAttrs("style").Matching(regexp.MustCompile(`^[-a-zA-Z0-9:;.%(), empx]+$`)).OnElements("span")
		p.AllowAttrs("aria-hidden").Matching(regexp.MustCompile(`^(?:true|false)$`)).OnElements("span")
		markdownSanitize = p
	})
	return markdownSanitize
}

// SanitizeHTML strips unsafe tags/attributes using MarkdownSanitizePolicy (UGC + KaTeX).
func SanitizeHTML(raw []byte) []byte {
	return MarkdownSanitizePolicy().SanitizeBytes(raw)
}

// MarkdownToSafeHTML converts markdown to HTML and sanitizes it for browser display.
// Use this (not MarkdownToHTML alone) before templ.Raw / template.HTML.
func MarkdownToSafeHTML(source []byte) ([]byte, error) {
	raw, err := MarkdownToHTML(source)
	if err != nil {
		return nil, err
	}
	return SanitizeHTML(raw), nil
}
