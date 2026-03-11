package components

import (
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

func HighlightText(text, query string) app.UI {
	if query == "" || text == "" {
		return app.Text(text)
	}

	// Find all case-insensitive matches
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var parts []app.UI
	lastIdx := 0

	for {
		idx := strings.Index(lowerText[lastIdx:], lowerQuery)
		if idx == -1 {
			// No more matches, add remaining text
			if lastIdx < len(text) {
				parts = append(parts, app.Text(text[lastIdx:]))
			}
			break
		}

		// Adjust index to absolute position
		absIdx := lastIdx + idx

		// Add text before match
		if absIdx > lastIdx {
			parts = append(parts, app.Text(text[lastIdx:absIdx]))
		}

		// Add highlighted match (preserve original case)
		matchEnd := absIdx + len(query)
		parts = append(parts, app.Mark().
			Class("bg-yellow-200 dark:bg-yellow-800 text-inherit rounded px-0.5").
			Text(text[absIdx:matchEnd]))

		lastIdx = matchEnd
	}

	if len(parts) == 0 {
		return app.Text(text)
	}

	return app.Span().Body(parts...)
}

func HighlightTextIf(text, query string, highlight bool) app.UI {
	if !highlight {
		return app.Text(text)
	}
	return HighlightText(text, query)
}
