package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var clipsListWebserviceRules = []webservice.Rule{
	webservice.Get("/clips", clipsListPage),
	webservice.Get("/clips/list", clipsListPartial),
	webservice.Post("/clips/:slug/visibility", clipSetVisibility),
}

// clipsListPage renders the authenticated clips browser under Integrate.
func clipsListPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := loadClipListItems(ctx.Context())
	if err != nil {
		flog.Error(fmt.Errorf("clipsListPage: %w", err))
		return ctx.Status(http.StatusInternalServerError).SendString(webMsg(ctx, "error.load.clips"))
	}
	ctx.Type("html")
	return pages.ClipsPage(ctx.Context(), items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// clipsListPartial returns the clips table fragment for HTMX refresh.
func clipsListPartial(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := loadClipListItems(ctx.Context())
	if err != nil {
		flog.Error(fmt.Errorf("clipsListPartial: %w", err))
		return renderErrorKey(ctx, "error.load.clips")
	}
	ctx.Type("html")
	return partials.ClipsTable(ctx.Context(), items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// clipSetVisibility toggles whether a clip is readable by anonymous visitors.
func clipSetVisibility(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	slug := strings.TrimSpace(ctx.Params("slug"))
	if slug == "" {
		return toastErrorKey(ctx, "error.clip.missing_slug")
	}
	isPublic := ctx.FormValue("is_public") == "true"

	if store.Database == nil || store.Database.GetClient() == nil {
		return toastErrorKey(ctx, "empty.store_unavailable")
	}
	row, err := store.ClipStoreFromDB().UpdateClipVisibility(context.Background(), slug, isPublic)
	if err != nil {
		flog.Error(fmt.Errorf("clipSetVisibility: %w", err))
		return toastErrorKey(ctx, "error.clip.update_visibility_failed")
	}
	if row == nil {
		return toastErrorKey(ctx, "clip.not_found.title")
	}

	ctx.Type("html")
	return partials.ClipRow(ctx.Context(), clipRowToListItem(row)).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func loadClipListItems(ctx context.Context) ([]partials.ClipListItem, error) {
	if store.Database == nil || store.Database.GetClient() == nil {
		return nil, fmt.Errorf("store not available")
	}
	rows, err := store.ClipStoreFromDB().ListClips(ctx, 200)
	if err != nil {
		return nil, err
	}
	return clipRowsToListItems(rows), nil
}

func clipRowsToListItems(rows []*gen.Clip) []partials.ClipListItem {
	items := make([]partials.ClipListItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, clipRowToListItem(row))
	}
	return items
}

func clipRowToListItem(row *gen.Clip) partials.ClipListItem {
	return partials.ClipListItem{
		Slug:        row.Slug,
		Title:       row.Title,
		Description: row.Description,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
		URL:         "/c/" + row.Slug,
		IsPublic:    row.IsPublic,
	}
}
