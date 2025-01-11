package bookmark

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"time"
)

var cronRules = []cron.Rule{
	{
		Name:  "bookmarks_tag",
		Scope: cron.CronScopeSystem,
		When:  "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			bookmarks, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			for _, bookmark := range bookmarks {
				if len(bookmark.Tags) > 0 {
					continue
				}
				tags, err := extractTags(ctx.Context(), bookmark)
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
				flog.Info("[bookmark] bookmark %s attach tags %v, result %v", bookmark.Id, tags, resp)
			}

			return nil
		},
	},
	{
		Name:  "bookmarks_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			bookmarks, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			bookmarkTotal := 0
			for _, bookmark := range bookmarks {
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
	{
		Name:  "bookmarks_search",
		Scope: cron.CronScopeSystem,
		When:  "*/5 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			bookmarks, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			for _, bookmark := range bookmarks {
				title := ""
				if bookmark.Content.BookmarkContentOneOf.Title.IsSet() &&
					bookmark.Content.BookmarkContentOneOf.Title.Get() != nil {
					title = *bookmark.Content.BookmarkContentOneOf.Title.Get()
				}
				summary := ""
				if bookmark.Summary.IsSet() &&
					bookmark.Summary.Get() != nil {
					summary = *bookmark.Summary.Get()
				}

				err := meilisearch.NewMeiliSearch().AddDocument(types.Document{
					SourceId:    bookmark.Id,
					Source:      hoarder.ID,
					Title:       title,
					Description: summary,
					Url:         fmt.Sprintf("/dashboard/preview/%s", bookmark.Id),
					Timestamp:   time.Now().Unix(),
				})
				if err != nil {
					flog.Warn("[search] add document error %v", err)
				}
			}

			return nil
		},
	},
}
