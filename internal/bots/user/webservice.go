package user

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/dashboard", dashboard),
	webservice.Get("/metrics", metrics),
	webservice.Get("/kanban", getKanban),
	webservice.Get("/bookmark", getBookmark),
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
func dashboard(ctx fiber.Ctx) error {
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
func metrics(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		stats.BotTotalStatsName:             rdb.GetMetricsInt64(stats.BotTotalStatsName),
		stats.BookmarkTotalStatsName:        rdb.GetMetricsInt64(stats.BookmarkTotalStatsName),
		stats.TorrentDownloadTotalStatsName: rdb.GetMetricsInt64(stats.TorrentDownloadTotalStatsName),
		stats.GiteaIssueTotalStatsName:      rdb.GetMetricsInt64(stats.GiteaIssueTotalStatsName),
		stats.ReaderUnreadTotalStatsName:    rdb.GetMetricsInt64(stats.ReaderUnreadTotalStatsName),
		stats.KanbanTaskTotalStatsName:      rdb.GetMetricsInt64(stats.KanbanTaskTotalStatsName),
		stats.MonitorUpTotalStatsName:       rdb.GetMetricsInt64(stats.MonitorUpTotalStatsName),
		stats.MonitorDownTotalStatsName:     rdb.GetMetricsInt64(stats.MonitorDownTotalStatsName),
		stats.DockerContainerTotalStatsName: rdb.GetMetricsInt64(stats.DockerContainerTotalStatsName),
	}))
}

// get user kanban list
//
//	@Summary	get user kanban list
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/user/kanban [get]
func getKanban(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to new client %w", err)
	}

	list, err := client.GetAllTasks(ctx.RequestCtx(), kanboard.DefaultProjectId, kanboard.Active)
	if err != nil {
		return fmt.Errorf("failed to get all tasks, %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// get user bookmark list
//
//	@Summary	get user bookmark list
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/user/bookmark [get]
func getBookmark(ctx fiber.Ctx) error {
	client := hoarder.GetClient()

	resp, err := client.GetAllBookmarks(&hoarder.BookmarksQuery{Limit: hoarder.MaxPageSize})
	if err != nil {
		return fmt.Errorf("failed to get all bookmarks, %w", err)
	}

	list := make([]hoarder.Bookmark, 0, 10)
	for i, item := range resp.Bookmarks {
		if item.Archived {
			continue
		}
		list = append(list, resp.Bookmarks[i])
	}

	return ctx.JSON(protocol.NewSuccessResponse(list))
}
