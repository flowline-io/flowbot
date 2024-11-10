package user

import (
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/gofiber/fiber/v2"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/dashboard", dashboard),
	webservice.Get("/metrics", metrics),
}

// dashboard show dashboard data
//
//	@Summary	Show dashboard
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/user/dashboard [get]
func dashboard(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "dashboard",
	}))
}

// metrics show metrics data
//
//	@Summary	Show metrics
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/user/widget [get]
func metrics(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		stats.BookmarkTotalStatsName: cache.GetInt64(stats.BookmarkTotalStatsName),
		stats.BotTotalStatsName:      cache.GetInt64(stats.BotTotalStatsName),
	}))
}
