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
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/user/dashboard [get]
func dashboard(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "dashboard",
	}))
}

// metrics show metrics data
//
//	@Summary	Show metrics
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/user/metrics [get]
func metrics(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		stats.BotTotalStatsName:             cache.GetInt64(stats.BotTotalStatsName),
		stats.BookmarkTotalStatsName:        cache.GetInt64(stats.BookmarkTotalStatsName),
		stats.TorrentDownloadTotalStatsName: cache.GetInt64(stats.TorrentDownloadTotalStatsName),
		stats.GiteaIssueTotalStatsName:      cache.GetInt64(stats.GiteaIssueTotalStatsName),
		stats.ReaderUnreadTotalStatsName:    cache.GetInt64(stats.ReaderUnreadTotalStatsName),
	}))
}
