package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var clipsListWebserviceRules = []webservice.Rule{
	webservice.Get("/clips", clipsListPage, route.WithNotAuth()),
	webservice.Get("/clips/list", clipsListPartial, route.WithNotAuth()),
}

// clipsListPage renders the authenticated clips browser under Integrate.
func clipsListPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := loadClipListItems(ctx.Context())
	if err != nil {
		flog.Error(fmt.Errorf("clipsListPage: %w", err))
		return ctx.Status(http.StatusInternalServerError).SendString("failed to load clips")
	}
	ctx.Type("html")
	return pages.ClipsPage(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// clipsListPartial returns the clips table fragment for HTMX refresh.
func clipsListPartial(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := loadClipListItems(ctx.Context())
	if err != nil {
		flog.Error(fmt.Errorf("clipsListPartial: %w", err))
		return renderError(ctx, "Failed to load clips")
	}
	ctx.Type("html")
	return partials.ClipsTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func loadClipListItems(ctx context.Context) ([]partials.ClipListItem, error) {
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		return nil, fmt.Errorf("store not available")
	}
	rows, err := store.NewClipStore(client).ListClips(ctx, 200)
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
		items = append(items, partials.ClipListItem{
			Slug:        row.Slug,
			Title:       row.Title,
			Description: row.Description,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   row.CreatedAt,
			URL:         "/c/" + row.Slug,
		})
	}
	return items
}
