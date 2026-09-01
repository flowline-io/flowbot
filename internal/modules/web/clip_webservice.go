package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

// clipPage renders GET /c/:slug. Public clips are readable anonymously; private clips
// require a web session and return 404 for anonymous visitors.
func clipPage(ctx fiber.Ctx) error {
	slug := ctx.Params("slug")
	if slug == "" {
		return ctx.Status(http.StatusBadRequest).SendString(webMsg(ctx, "error.clip.missing_slug"))
	}

	authed := isAuthenticated(ctx)

	if store.Database == nil || store.Database.GetClient() == nil {
		return ctx.Status(http.StatusInternalServerError).SendString(webMsg(ctx, "empty.store_unavailable"))
	}
	clipStore := store.ClipStoreFromDB()

	row, err := clipStore.GetClipBySlug(context.Background(), slug)
	if err != nil {
		flog.Error(fmt.Errorf("clipPage: GetClipBySlug: %w", err))
		return ctx.Status(http.StatusInternalServerError).SendString(webMsg(ctx, "error.clip.load_failed"))
	}

	data := pages.ClipPageData{Slug: slug}

	if row == nil || (!authed && !row.IsPublic) {
		data.NotFound = true
		data.Title = webMsg(ctx, "clip.not_found.title")
		data.Description = webMsg(ctx, "clip.not_found.meta")
		ctx.Type("html")
		ctx.Status(http.StatusNotFound)
		return pages.ClipPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	data.Title = row.Title
	data.Description = row.Description
	data.CreatedAt = row.CreatedAt
	data.WordCount = utils.WordCount(row.Content)
	data.ContentMD = row.Content

	html, mdErr := utils.MarkdownToSafeHTML([]byte(row.Content))
	if mdErr != nil {
		flog.Error(fmt.Errorf("clipPage: MarkdownToSafeHTML: %w", mdErr))
		html = []byte("<pre>failed to render markdown</pre>")
	}
	data.BodyHTML = string(html)

	ctx.Type("html")
	return pages.ClipPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}
