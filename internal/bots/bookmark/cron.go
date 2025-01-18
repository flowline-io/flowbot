package bookmark

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

const (
	defaultProjectId = 1
	defaultPriority  = 2
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
				title := bookmark.GetTitle()
				summary := bookmark.GetSummary()
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
	{
		Name:  "bookmarks_task",
		Scope: cron.CronScopeUser,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			bookmarks, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			for _, bookmark := range bookmarks {
				if bookmark.Archived {
					continue
				}

				// filter
				ok, err := cache.UniqueString(ctx.Context(), "bookmarks:task:filter", bookmark.Id)
				if err != nil {
					flog.Error(fmt.Errorf("cron bookmarks_task unique error %w", err))
					continue
				}
				if !ok {
					continue
				}

				// create task
				title := bookmark.GetContent().BookmarkContentOneOf.GetTitle()
				err = event.BotEventFire(ctx, types.TaskCreateBotEventID, types.KV{
					"title":       title,
					"project_id":  defaultProjectId,
					"priority":    defaultPriority,
					"reference":   fmt.Sprintf("%s:%s", hoarder.ID, bookmark.Id),
					"description": fmt.Sprintf("%s/dashboard/preview/%s", config.App.Search.UrlBaseMap[hoarder.ID], bookmark.Id),
					"tags": []string{
						Name,
						hoarder.ID,
					},
				})
				if err != nil {
					flog.Error(fmt.Errorf("cron bookmarks_task event fire error %w", err))
					continue
				}
			}

			return nil
		},
	},
}
