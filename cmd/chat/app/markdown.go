package app

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"charm.land/glamour/v2"
)

const (
	debounceShort       = 100 * time.Millisecond
	debounceMid         = 200 * time.Millisecond
	debounceLong        = 500 * time.Millisecond
	debounceCodeLine    = 400 * time.Millisecond
	debounceCodeMidLine = 600 * time.Millisecond
)

var (
	glamourMu       sync.Mutex
	glamourWidth    int
	glamourRenderer *glamour.TermRenderer
	ansiRe          = regexp.MustCompile(`\x1b\[[0-9;]*[ -/]*[@-~]`)
)

// RenderDebounce returns how long to wait before re-rendering markdown for buf.
// Shorter buffers refresh quickly; long replies and in-progress code fences back off
// to avoid re-running glamour on every streaming delta.
func RenderDebounce(buf string) time.Duration {
	n := len(buf)
	inCode := inOpenCodeFence(buf)

	if inCode && n > 0 && buf[n-1] != '\n' {
		switch {
		case n < 1000:
			return debounceCodeMidLine
		case n <= 5000:
			return debounceLong
		default:
			return debounceLong + 200*time.Millisecond
		}
	}
	if inCode {
		return debounceCodeLine
	}

	switch {
	case n < 1000:
		return debounceShort
	case n <= 5000:
		return debounceMid
	default:
		return debounceLong
	}
}

func inOpenCodeFence(buf string) bool {
	count := strings.Count(buf, "```")
	return count%2 == 1
}

// RenderMarkdown renders assistant markdown for terminal display via glamour.
func RenderMarkdown(source string, width int) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	if width <= 0 {
		width = 80
	}

	r, err := glamourRendererForWidth(width)
	if err != nil {
		return source
	}
	out, err := r.Render(source)
	if err != nil {
		return source
	}
	return strings.TrimRight(out, "\n")
}

func glamourRendererForWidth(width int) (*glamour.TermRenderer, error) {
	glamourMu.Lock()
	defer glamourMu.Unlock()
	if glamourRenderer != nil && glamourWidth == width {
		return glamourRenderer, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("tokyo-night"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	glamourRenderer = r
	glamourWidth = width
	return r, nil
}

// stripANSI removes terminal escape sequences for test assertions.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
