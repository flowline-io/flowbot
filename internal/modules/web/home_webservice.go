package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var homeWebserviceRules = []webservice.Rule{
	webservice.Get("/home", homePage, route.WithNotAuth()),
	webservice.Get("/home/token-usage", homeTokenUsage, route.WithNotAuth()),
}

func homePage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.HomePage().Render(context.Background(), ctx.Response().BodyWriter())
}

func homeTokenUsage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	uid := getUID(c)
	if uid == "" {
		return redirectToLogin(c)
	}

	groupBy, err := types.NormalizeTokenUsageGroupBy(c.Query("groupBy", ""))
	if err != nil {
		return invalidTokenUsageRequest(c, err)
	}

	since, until, activeRange, rangeLabel, err := types.ResolveTokenUsageRange(
		c.Query("range", ""),
		c.Query("since", ""),
		c.Query("until", ""),
		time.Now().UTC(),
	)
	if err != nil {
		return invalidTokenUsageRequest(c, err)
	}

	usageStore := store.NewLLMUsageStoreFromDatabase()
	if usageStore == nil {
		return types.Errorf(types.ErrInternal, "store not available")
	}

	stats, err := usageStore.TokenUsageStats(c.Context(), uid, since, until, groupBy)
	if err != nil {
		return types.Errorf(types.ErrInternal, "token usage stats: %v", err)
	}
	stats.RangeLabel = rangeLabel
	stats.ActiveRange = activeRange
	stats.GroupBy = groupBy

	accept := c.Get("Accept", "")
	if strings.Contains(accept, "application/json") {
		return c.JSON(stats)
	}
	c.Type("html")
	return partials.TokenUsage(stats).Render(context.Background(), c.Response().BodyWriter())
}

func invalidTokenUsageRequest(c fiber.Ctx, err error) error {
	if errors.Is(err, types.ErrInvalidArgument) {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	return c.Status(http.StatusBadRequest).SendString(err.Error())
}
