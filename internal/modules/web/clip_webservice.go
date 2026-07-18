package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	abilityclip "github.com/flowline-io/flowbot/pkg/capability/clip"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

// clipPage renders GET /c/:slug — anonymous visitors get title/description meta;
// authenticated web users see the full markdown body.
func clipPage(ctx fiber.Ctx) error {
	slug := ctx.Params("slug")
	if slug == "" {
		return ctx.Status(http.StatusBadRequest).SendString("missing slug")
	}

	authed := isAuthenticated(ctx)

	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		return ctx.Status(http.StatusInternalServerError).SendString("store not available")
	}
	clipStore := store.NewClipStore(client)

	row, err := clipStore.GetClipBySlug(context.Background(), slug)
	if err != nil {
		flog.Error(fmt.Errorf("clipPage: GetClipBySlug: %w", err))
		return ctx.Status(http.StatusInternalServerError).SendString("failed to load clip")
	}

	loginURL := "/service/web/login?next=" + url.QueryEscape("/c/"+slug)
	data := pages.ClipPageData{
		Slug:     slug,
		Authed:   authed,
		LoginURL: loginURL,
	}

	if row == nil {
		data.NotFound = true
		data.Title = "Clip not found"
		data.Description = "This clip does not exist or was removed."
		ctx.Type("html")
		ctx.Status(http.StatusNotFound)
		return pages.ClipPage(data).Render(context.Background(), ctx.Response().BodyWriter())
	}

	data.Title = row.Title
	data.Description = row.Description
	data.CreatedAt = row.CreatedAt
	data.WordCount = abilityclip.WordCount(row.Content)
	data.ContentMD = row.Content

	if authed {
		html, mdErr := utils.MarkdownToSafeHTML([]byte(row.Content))
		if mdErr != nil {
			flog.Error(fmt.Errorf("clipPage: MarkdownToSafeHTML: %w", mdErr))
			html = []byte("<pre>failed to render markdown</pre>")
		}
		data.BodyHTML = string(html)
	}

	ctx.Type("html")
	return pages.ClipPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}
