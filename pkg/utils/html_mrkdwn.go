package utils

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	htmlBreakTags = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlBlockEnd  = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr)>`)
	htmlAnchorTag = regexp.MustCompile(`(?is)<a\s+[^>]*href\s*=\s*["']([^"']+)["'][^>]*>(.*?)</a>`)
	htmlBoldTag   = regexp.MustCompile(`(?is)<(?:b|strong)\b[^>]*>(.*?)</(?:b|strong)>`)
	htmlItalicTag = regexp.MustCompile(`(?is)<(?:i|em)\b[^>]*>(.*?)</(?:i|em)>`)
	htmlCodeTag   = regexp.MustCompile(`(?is)<code\b[^>]*>(.*?)</code>`)
	htmlListItem  = regexp.MustCompile(`(?is)<li\b[^>]*>`)
	htmlAnyTag    = regexp.MustCompile(`(?is)<[^>]+>`)
	htmlScriptTag = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
)

// HTMLToMrkdwn converts common HTML into Slack mrkdwn plain text.
// Unsupported tags are stripped; entities are unescaped.
func HTMLToMrkdwn(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	s := htmlScriptTag.ReplaceAllString(raw, "")
	s = htmlBreakTags.ReplaceAllString(s, "\n")
	s = htmlBlockEnd.ReplaceAllString(s, "\n")

	var links []string
	s = htmlAnchorTag.ReplaceAllStringFunc(s, func(m string) string {
		parts := htmlAnchorTag.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		idx := len(links)
		links = append(links, fmt.Sprintf("<%s|%s>", parts[1], parts[2]))
		return fmt.Sprintf("\x00LINK%d\x00", idx)
	})

	s = htmlBoldTag.ReplaceAllString(s, "*${1}*")
	s = htmlItalicTag.ReplaceAllString(s, "_${1}_")
	s = htmlCodeTag.ReplaceAllString(s, "`${1}`")
	s = htmlListItem.ReplaceAllString(s, "• ")
	s = htmlAnyTag.ReplaceAllString(s, "")

	for i, link := range links {
		s = strings.ReplaceAll(s, fmt.Sprintf("\x00LINK%d\x00", i), link)
	}

	s = html.UnescapeString(s)
	return strings.TrimSpace(collapseBlankLines(s))
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
