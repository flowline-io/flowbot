// Package pages provides full-page Templ views.
package pages

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

// clipPageTitle returns the HTML document title for a clip page.
func clipPageTitle(ctx context.Context, d ClipPageData) string {
	if d.NotFound {
		return i18n.T(ctx, "clip.not_found.title")
	}
	if d.Title == "" {
		return i18n.T(ctx, "clip.default_title")
	}
	return d.Title
}

// formatClipMeta formats created-at and word-count for the clip subtitle line.
func formatClipMeta(ctx context.Context, createdAt time.Time, wordCount int) string {
	if createdAt.IsZero() {
		if wordCount <= 0 {
			return ""
		}
		return i18n.TData(ctx, "clip.meta.words_only", map[string]any{"Count": wordCount})
	}
	stamp := createdAt.UTC().Format("Jan 2, 2006, 3:04 PM UTC")
	if wordCount <= 0 {
		return i18n.TData(ctx, "clip.meta.date_only", map[string]any{"Date": stamp})
	}
	return i18n.TData(ctx, "clip.meta.date_words", map[string]any{"Date": stamp, "Count": wordCount})
}
