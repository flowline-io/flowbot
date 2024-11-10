package bookmark

import (
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name: "bookmarks_tag",
		When: "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			resp, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			for _, bookmark := range resp.Bookmarks {
				if len(bookmark.Tags) > 0 {
					continue
				}
				tags, err := extractTags(ctx.Context(), bookmark.Title)
				if err != nil {
					flog.Error(err)
				}
				if len(tags) == 0 {
					continue
				}

				resp, err := client.AttachTagsToBookmark(bookmark.Id, tags)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[bookmark] bookmark %s attach tags %v,esult %v", bookmark.Id, tags, resp.Attached)
			}

			return nil
		},
	},
	{
		Name: "bookmarks_metrics",
		When: "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			resp, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			bookmarkTotal := 0
			for _, bookmark := range resp.Bookmarks {
				if bookmark.Archived {
					continue
				}
				bookmarkTotal++
			}
			stats.BookmarkTotalCounter().Set(uint64(bookmarkTotal))
			cache.SetInt64(stats.BookmarkTotalStatsName, int64(bookmarkTotal))

			return nil
		},
	},
}
