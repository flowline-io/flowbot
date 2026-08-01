package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	htmlBreakTags     = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlHrTag         = regexp.MustCompile(`(?i)<hr\s*/?>`)
	htmlBlockEnd      = regexp.MustCompile(`(?i)</(p|div)>`)
	htmlAnchorTag     = regexp.MustCompile(`(?is)<a\s+[^>]*href\s*=\s*["']([^"']+)["'][^>]*>(.*?)</a>`)
	htmlImgTag        = regexp.MustCompile(`(?is)<img\b([^>]*)>`)
	htmlImgSrcAttr    = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']([^"']+)["']`)
	htmlImgAltAttr    = regexp.MustCompile(`(?i)\balt\s*=\s*["']([^"']*)["']`)
	htmlBoldTag       = regexp.MustCompile(`(?is)<(?:b|strong)\b[^>]*>(.*?)</(?:b|strong)>`)
	htmlItalicTag     = regexp.MustCompile(`(?is)<(?:i|em)\b[^>]*>(.*?)</(?:i|em)>`)
	htmlStrikeTag     = regexp.MustCompile(`(?is)<(?:del|s|strike)\b[^>]*>(.*?)</(?:del|s|strike)>`)
	htmlHeadingTag    = regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)
	htmlListItemFull  = regexp.MustCompile(`(?is)<li\b[^>]*>(.*?)</li>`)
	htmlTableRowTag   = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	htmlTableCellTag  = regexp.MustCompile(`(?is)<t[hd]\b[^>]*>(.*?)</t[hd]>`)
	htmlPreCodeTag    = regexp.MustCompile(`(?is)<pre\b[^>]*>\s*<code\b[^>]*>(.*?)</code>\s*</pre>`)
	htmlPreTag        = regexp.MustCompile(`(?is)<pre\b[^>]*>(.*?)</pre>`)
	htmlCodeTag       = regexp.MustCompile(`(?is)<code\b[^>]*>(.*?)</code>`)
	htmlCheckboxInput = regexp.MustCompile(`(?is)<input\b[^>]*type\s*=\s*["']checkbox["'][^>]*>`)
	htmlAnyTag        = regexp.MustCompile(`(?is)<[^>]+>`)
	htmlScriptTag     = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	htmlListOpenTag       = regexp.MustCompile(`(?i)<(ul|ol)\b[^>]*>`)
	htmlBlockquoteOpenTag = regexp.MustCompile(`(?i)<blockquote\b[^>]*>`)
	mrkdwnHRLine          = regexp.MustCompile(`^\s*(?:-{3,}|\*{3,}|_{3,})\s*$`)
)

var listBullets = []string{"• ", "◦ ", "▪ "}

// MarkdownToMrkdwn converts GitHub-flavored markdown into Slack mrkdwn.
// On conversion failure it returns the original input unchanged.
func MarkdownToMrkdwn(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	// Intermediate HTML only; tags are stripped to mrkdwn, so skip browser sanitizer
	// to preserve task-list checkboxes and other structural markers.
	htmlBytes, err := MarkdownToHTML([]byte(raw))
	if err != nil {
		return raw
	}
	return HTMLToMrkdwn(string(htmlBytes))
}

// HTMLToMrkdwn converts common HTML into Slack mrkdwn plain text.
// Unsupported tags are stripped; entities are unescaped.
func HTMLToMrkdwn(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	s := htmlScriptTag.ReplaceAllString(raw, "")
	s = htmlBreakTags.ReplaceAllString(s, "\n")
	s = htmlHrTag.ReplaceAllString(s, "---\n")
	s = htmlBlockEnd.ReplaceAllString(s, "\n")

	var links []string
	s = replaceAnchors(s, &links)
	s = replaceImages(s, &links)
	s = replaceInlineMarkup(s)
	s = replaceCodeBlocks(s)
	s = replaceHeadings(s)
	s = replaceBlockquotes(s)
	s = replaceLists(s)
	s = replaceTableRows(s)
	s = htmlAnyTag.ReplaceAllString(s, "")

	for i, link := range links {
		s = strings.ReplaceAll(s, fmt.Sprintf("\x00LINK%d\x00", i), link)
	}

	s = html.UnescapeString(s)
	return strings.TrimSpace(collapseBlankLines(s))
}

// IsMrkdwnHorizontalRule reports whether a line is an HR marker for Block Kit dividers.
func IsMrkdwnHorizontalRule(line string) bool {
	return mrkdwnHRLine.MatchString(line)
}

func replaceAnchors(s string, links *[]string) string {
	return htmlAnchorTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlAnchorTag.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		href := strings.TrimSpace(parts[1])
		text := strings.TrimSpace(stripTags(parts[2]))
		text = html.UnescapeString(text)
		return stashLink(links, slackLink(href, text))
	})
}

func replaceImages(s string, links *[]string) string {
	return htmlImgTag.ReplaceAllStringFunc(s, func(m string) string {
		attrs := htmlImgTag.FindStringSubmatch(m)
		if len(attrs) != 2 {
			return ""
		}
		srcParts := htmlImgSrcAttr.FindStringSubmatch(attrs[1])
		if len(srcParts) != 2 {
			return ""
		}
		src := strings.TrimSpace(srcParts[1])
		alt := ""
		if altParts := htmlImgAltAttr.FindStringSubmatch(attrs[1]); len(altParts) == 2 {
			alt = strings.TrimSpace(html.UnescapeString(altParts[1]))
		}
		return stashLink(links, slackLink(src, alt))
	})
}

func replaceInlineMarkup(s string) string {
	s = htmlBoldTag.ReplaceAllString(s, "*${1}*")
	s = htmlItalicTag.ReplaceAllString(s, "_${1}_")
	return htmlStrikeTag.ReplaceAllString(s, "~${1}~")
}

func replaceCodeBlocks(s string) string {
	s = htmlPreCodeTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlPreCodeTag.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		return slackCodeFence(parts[1])
	})
	s = htmlPreTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlPreTag.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		return slackCodeFence(parts[1])
	})
	return htmlCodeTag.ReplaceAllString(s, "`${1}`")
}

func replaceHeadings(s string) string {
	return htmlHeadingTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlHeadingTag.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		inner := strings.TrimSpace(stripTags(parts[1]))
		if inner == "" {
			return ""
		}
		return "*" + inner + "*\n"
	})
}

func replaceBlockquotes(s string) string {
	for {
		start, end, inner, ok := findInnermostBlock(s, "blockquote")
		if !ok {
			return s
		}
		rendered := renderBlockquote(inner)
		s = s[:start] + rendered + s[end:]
	}
}

func renderBlockquote(inner string) string {
	inner = strings.TrimSpace(collapseBlankLines(stripTags(inner)))
	if inner == "" {
		return ""
	}
	lines := strings.Split(inner, "\n")
	for i, line := range lines {
		lines[i] = "> " + strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n") + "\n"
}

func replaceLists(s string) string {
	for {
		start, end, tag, inner, ok := findInnermostList(s)
		if !ok {
			return s
		}
		depth := enclosingListDepth(s, start)
		ordered := strings.EqualFold(tag, "ol")
		rendered := renderListItems(inner, ordered, depth)
		// Keep nested list content on its own lines when HTML is compacted.
		if depth > 0 && rendered != "" && !strings.HasPrefix(rendered, "\n") {
			rendered = "\n" + rendered
		}
		s = s[:start] + rendered + s[end:]
	}
}

func findInnermostList(s string) (start, end int, tag, inner string, ok bool) {
	matches := htmlListOpenTag.FindAllStringSubmatchIndex(s, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		openStart, openEnd := m[0], m[1]
		tagName := s[m[2]:m[3]]
		_, closeEnd, body, found := findBalancedClose(s, openEnd, tagName)
		if !found {
			continue
		}
		if htmlListOpenTag.MatchString(body) {
			continue
		}
		return openStart, closeEnd, tagName, body, true
	}
	return 0, 0, "", "", false
}

func findInnermostBlock(s, tag string) (start, end int, inner string, ok bool) {
	var openRe *regexp.Regexp
	switch strings.ToLower(tag) {
	case "blockquote":
		openRe = htmlBlockquoteOpenTag
	default:
		return 0, 0, "", false
	}
	matches := openRe.FindAllStringIndex(s, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		openStart, openEnd := matches[i][0], matches[i][1]
		_, closeEnd, body, found := findBalancedClose(s, openEnd, tag)
		if !found {
			continue
		}
		if openRe.MatchString(body) {
			continue
		}
		return openStart, closeEnd, body, true
	}
	return 0, 0, "", false
}

func findBalancedClose(s string, contentStart int, tag string) (closeStart, closeEnd int, inner string, ok bool) {
	lower := strings.ToLower(s)
	tagLower := strings.ToLower(tag)
	openPrefix := "<" + tagLower
	closeTag := "</" + tagLower + ">"
	depth := 1
	i := contentStart
	for i < len(s) && depth > 0 {
		nextOpen := strings.Index(lower[i:], openPrefix)
		nextClose := strings.Index(lower[i:], closeTag)
		if nextClose < 0 {
			return 0, 0, "", false
		}
		if nextOpen >= 0 && nextOpen < nextClose {
			// Ensure it is a real open tag for this element (ul/ol/blockquote).
			abs := i + nextOpen
			after := abs + len(openPrefix)
			if after < len(s) {
				c := s[after]
				if c == '>' || c == ' ' || c == '\t' || c == '\n' || c == '/' {
					depth++
					i = after
					continue
				}
			}
			i = abs + 1
			continue
		}
		absClose := i + nextClose
		depth--
		if depth == 0 {
			return absClose, absClose + len(closeTag), s[contentStart:absClose], true
		}
		i = absClose + len(closeTag)
	}
	return 0, 0, "", false
}

func enclosingListDepth(s string, pos int) int {
	depth := 0
	lower := strings.ToLower(s[:pos])
	i := 0
	for i < len(lower) {
		j := strings.IndexByte(lower[i:], '<')
		if j < 0 {
			break
		}
		i += j
		rest := lower[i:]
		switch {
		case strings.HasPrefix(rest, "<ul"), strings.HasPrefix(rest, "<ol"):
			after := 3
			if after < len(rest) {
				c := rest[after]
				if c == '>' || c == ' ' || c == '\t' || c == '\n' || c == '/' {
					depth++
				}
			}
		case strings.HasPrefix(rest, "</ul>"), strings.HasPrefix(rest, "</ol>"):
			depth--
		}
		i++
	}
	if depth < 0 {
		return 0
	}
	return depth
}

func replaceTableRows(s string) string {
	return htmlTableRowTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlTableRowTag.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		cells := htmlTableCellTag.FindAllStringSubmatch(parts[1], -1)
		if len(cells) == 0 {
			return ""
		}
		cols := make([]string, 0, len(cells))
		for _, cell := range cells {
			cols = append(cols, strings.TrimSpace(stripTags(cell[1])))
		}
		return strings.Join(cols, " | ") + "\n"
	})
}

func renderListItems(inner string, ordered bool, depth int) string {
	items := htmlListItemFull.FindAllStringSubmatch(inner, -1)
	if len(items) == 0 {
		return ""
	}
	indent := strings.Repeat("    ", depth)
	out := make([]string, 0, len(items))
	for i, item := range items {
		body := item[1]
		prefix := indent + listBullets[depth%len(listBullets)]
		if ordered {
			prefix = indent + fmt.Sprintf("%d. ", i+1)
		}
		if htmlCheckboxInput.MatchString(body) {
			checked := strings.Contains(strings.ToLower(body), "checked")
			body = htmlCheckboxInput.ReplaceAllString(body, "")
			if checked {
				prefix += "✅ "
			} else {
				prefix += "⬜ "
			}
		}
		text := strings.TrimSpace(stripTags(body))
		if text == "" {
			continue
		}
		lines := strings.Split(text, "\n")
		out = append(out, prefix+strings.TrimSpace(lines[0]))
		for _, line := range lines[1:] {
			trimmed := strings.TrimRight(line, " \t")
			if strings.TrimSpace(trimmed) == "" {
				continue
			}
			// Nested list lines already carry their own indentation.
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

func slackLink(href, text string) string {
	href = strings.TrimSpace(href)
	if !isSlackableURL(href) {
		if text != "" {
			return text
		}
		if decoded, err := url.PathUnescape(strings.TrimPrefix(href, "#")); err == nil && decoded != "" {
			return decoded
		}
		return href
	}
	if text == "" || text == href {
		return fmt.Sprintf("<%s>", href)
	}
	return fmt.Sprintf("<%s|%s>", href, text)
}

func isSlackableURL(href string) bool {
	h := strings.ToLower(href)
	return strings.HasPrefix(h, "http://") ||
		strings.HasPrefix(h, "https://") ||
		strings.HasPrefix(h, "mailto:") ||
		strings.HasPrefix(h, "tel:") ||
		strings.HasPrefix(h, "slack://")
}

func stashLink(links *[]string, rendered string) string {
	idx := len(*links)
	*links = append(*links, rendered)
	return fmt.Sprintf("\x00LINK%d\x00", idx)
}

func stripTags(s string) string {
	return htmlAnyTag.ReplaceAllString(s, "")
}

func slackCodeFence(body string) string {
	return "```\n" + strings.TrimRight(body, "\n") + "\n```"
}

func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		blank := strings.TrimSpace(trimmed) == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, trimmed)
		prevBlank = blank
	}
	return strings.Join(out, "\n")
}
