package bookmark

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/agents"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "bookmarks_tag",
		Scope: cron.CronScopeSystem,
		When:  "0 2 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if !agents.AgentEnabled(agents.AgentExtractTags) {
				flog.Info("agent extract tags disabled")
				return nil
			}

			client := hoarder.GetClient()
			resp, err := client.GetAllBookmarks(nil)
			if err != nil {
				flog.Error(err)
				return nil
			}

			for _, bookmark := range resp.Bookmarks {
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
					continue
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
			client := hoarder.GetClient()
			resp, err := client.GetAllBookmarks(nil)
			if err != nil {
				flog.Error(err)
				return nil
			}

			bookmarkTotal := 0
			for _, bookmark := range resp.Bookmarks {
				if bookmark.Archived {
					continue
				}
				bookmarkTotal++
			}
			stats.BookmarkTotalCounter().Set(uint64(bookmarkTotal))
			rdb.SetMetricsInt64(stats.BookmarkTotalStatsName, int64(bookmarkTotal))

			return nil
		},
	},
	{
		Name:  "bookmarks_search",
		Scope: cron.CronScopeSystem,
		When:  "*/5 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			client := hoarder.GetClient()
			resp, err := client.GetAllBookmarks(nil)
			if err != nil {
				flog.Error(err)
				return nil
			}

			for _, bookmark := range resp.Bookmarks {
				title := bookmark.GetTitle()
				summary := bookmark.GetSummary()
				err := search.Instance.AddDocument(types.Document{
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
			client := hoarder.GetClient()
			resp, err := client.GetAllBookmarks(nil)
			if err != nil {
				flog.Error(err)
				return nil
			}

			for _, bookmark := range resp.Bookmarks {
				if bookmark.Archived {
					continue
				}

				title := bookmark.Content.Title
				if title == nil {
					continue
				}

				// filter
				ok, err := rdb.BloomUniqueString(ctx.Context(), "bookmarks:task:filter", bookmark.Id)
				if err != nil {
					flog.Error(fmt.Errorf("cron bookmarks_task unique error %w", err))
					continue
				}
				if !ok {
					continue
				}

				// create task
				err = event.BotEventFire(ctx, types.TaskCreateBotEventID, types.KV{
					"title":       title,
					"project_id":  kanboard.DefaultProjectId,
					"priority":    kanboard.DefaultPriority,
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
	{
		Name:  "bookmarks_tag_merge",
		Scope: cron.CronScopeSystem,
		When:  "0 2 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if !agents.AgentEnabled(agents.AgentSimilarTags) {
				flog.Info("agent similar tags disabled")
				return nil
			}

			// Get all tags
			client := hoarder.GetClient()
			tags, err := client.GetAllTags()
			if err != nil {
				flog.Error(fmt.Errorf("get all tags error: %w", err))
				return nil
			}

			// Analyze similar tags using a large model
			tagStrings := convertTagsToStrings(tags)
			ctx.SetTimeout(10 * time.Minute)
			similarTags, err := analyzeSimilarTags(ctx.Context(), tagStrings)
			if err != nil {
				flog.Error(fmt.Errorf("analyze similar tags error: %w", err))
				return nil
			}

			var nextCursor string
			for {
				// Get all bookmarks
				resp, err := client.GetAllBookmarks(&hoarder.BookmarksQuery{Limit: hoarder.MaxPageSize, Cursor: nextCursor})
				if err != nil {
					flog.Error(fmt.Errorf("get all bookmarks error: %w", err))
					return nil
				}

				// Replace tags in bookmarks
				for _, bookmark := range resp.Bookmarks {
					oldTagStrings := convertBookmarkTagsToStrings(bookmark.Tags)
					newTagStrings := replaceSimilarTags(oldTagStrings, similarTags)
					if len(newTagStrings) == 0 || sliceEqual(oldTagStrings, newTagStrings) {
						continue
					}

					flog.Info("[bookmark] %s update tags from %v to %v", bookmark.Id, oldTagStrings, newTagStrings)

					// remove all old tags
					_, err = client.DetachTagsToBookmark(bookmark.Id, oldTagStrings)
					if err != nil {
						flog.Error(fmt.Errorf("detach bookmark %s tags error: %w", bookmark.Id, err))
						continue
					}

					// add new tags
					_, err = client.AttachTagsToBookmark(bookmark.Id, newTagStrings)
					if err != nil {
						flog.Error(fmt.Errorf("attach bookmark %s tags error: %w", bookmark.Id, err))
						continue
					}
				}
				nextCursor = resp.NextCursor
				if nextCursor == "" {
					break
				}
			}

			return nil
		},
	},
}
