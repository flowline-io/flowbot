// Package pages provides full-page Templ views.
package pages

import (
	"fmt"
	"time"
)

// clipPageTitle returns the HTML document title for a clip page.
func clipPageTitle(d ClipPageData) string {
	if d.NotFound {
		return "Clip not found"
	}
	if d.Title == "" {
		return "Clip"
	}
	return d.Title
}

// formatClipMeta formats created-at and word-count for the clip subtitle line.
func formatClipMeta(createdAt time.Time, wordCount int) string {
	if createdAt.IsZero() {
		if wordCount <= 0 {
			return ""
		}
		return fmt.Sprintf("%d words", wordCount)
	}
	stamp := createdAt.UTC().Format("Jan 2, 2006, 3:04 PM UTC")
	if wordCount <= 0 {
		return stamp
	}
	return fmt.Sprintf("%s · %d words", stamp, wordCount)
}
