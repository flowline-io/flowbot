// Package partials provides reusable Templ fragments for HTMX responses.
package partials

import "time"

// formatClipListTime formats a clip created-at timestamp for the list table.
func formatClipListTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}
