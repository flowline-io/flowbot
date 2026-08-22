package web

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func lifeStatsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifeStatsPage(pages.LifeStatsShellData{
		PendingCount: len(pending),
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifeStatsPanel(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	tz := string(ctx.Query("tz"))
	page, err := lifeService().GetStatsPage(context.Background(), uid, tz)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return partials.LifeStatsPanel(partials.LifeStatsFromPage(ctx.Context(), page)).Render(
		ctx.Context(),
		ctx.Response().BodyWriter(),
	)
}
