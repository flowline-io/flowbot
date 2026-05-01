package bookmark

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/llm"
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
			if !llm.AgentEnabled(llm.AgentExtractTags) {
				flog.Info("agent extract tags disabled")
				return nil
			}

			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "list", map[string]any{})
			if err != nil {
				flog.Error(err)
				return nil
			}

			bookmarks, _ := res.Data.([]*ability.Bookmark)
			for _, bookmark := range bookmarks {
				if len(bookmark.Tags) > 0 {
					continue
				}
				tags, err := extractTags(ctx.Context(), bookmark.URL, bookmark.Title)
				if err != nil {
					flog.Error(err)
				}
				if len(tags) == 0 {
					continue
				}

				_, err = ability.Invoke(ctx.Context(), hub.CapBookmark, "attach_tags", map[string]any{
					"id":   bookmark.ID,
					"tags": tags,
				})
				if err != nil {
					flog.Error(err)
					continue
				}
				flog.Info("[bookmark] bookmark %s attach tags %v", bookmark.ID, tags)
			}

			return nil
		},
	},
	{
		Name:  "bookmarks_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "list", map[string]any{})
			if err != nil {
				flog.Error(err)
				return nil
			}

			bookmarkTotal := 0
			bookmarks, _ := res.Data.([]*ability.Bookmark)
			for _, bookmark := range bookmarks {
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
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "list", map[string]any{})
			if err != nil {
				flog.Error(err)
				return nil
			}

			bookmarks, _ := res.Data.([]*ability.Bookmark)
			for _, bookmark := range bookmarks {
				err := search.Instance.AddDocument(types.Document{
					SourceId:    bookmark.ID,
					Source:      "karakeep",
					Title:       bookmark.Title,
					Description: bookmark.Summary,
					Url:         fmt.Sprintf("/dashboard/preview/%s", bookmark.ID),
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
		Description: "Creates kanban tasks for new bookmarks. " +
			"Prefer using a pipeline config (trigger: bookmark.created) for this cross-service behavior.",
		Action: func(ctx types.Context) []types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "list", map[string]any{})
			if err != nil {
				flog.Error(err)
				return nil
			}

			bookmarks, _ := res.Data.([]*ability.Bookmark)
			for _, bookmark := range bookmarks {
				if bookmark.Archived {
					continue
				}
				if bookmark.Title == "" {
					continue
				}

				ok, err := rdb.BloomUniqueString(ctx.Context(), "bookmarks:task:filter", bookmark.ID)
				if err != nil {
					flog.Error(fmt.Errorf("cron bookmarks_task unique error %w", err))
					continue
				}
				if !ok {
					continue
				}

				err = event.BotEventFire(ctx, types.TaskCreateBotEventID, types.KV{
					"title":       bookmark.Title,
					"project_id":  int64(1),
					"priority":    int64(2),
					"reference":   fmt.Sprintf("karakeep:%s", bookmark.ID),
					"description": fmt.Sprintf("%s/dashboard/preview/%s", config.App.Search.UrlBaseMap["karakeep"], bookmark.ID),
					"tags": []string{
						Name,
						"karakeep",
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
			if !llm.AgentEnabled(llm.AgentSimilarTags) {
				flog.Info("agent similar tags disabled")
				return nil
			}

			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "list", map[string]any{})
			if err != nil {
				flog.Error(fmt.Errorf("get all bookmarks error: %w", err))
				return nil
			}

			bookmarks, _ := res.Data.([]*ability.Bookmark)

			tagSet := make(map[string]struct{})
			for _, bookmark := range bookmarks {
				for _, tag := range bookmark.Tags {
					tagSet[tag] = struct{}{}
				}
			}
			tagList := make([]string, 0, len(tagSet))
			for tag := range tagSet {
				tagList = append(tagList, tag)
			}

			ctx.SetTimeout(10 * time.Minute)
			similarTags, err := analyzeSimilarTags(ctx.Context(), tagList)
			if err != nil {
				flog.Error(fmt.Errorf("analyze similar tags error: %w", err))
				return nil
			}

			for _, bookmark := range bookmarks {
				oldTags := bookmark.Tags
				newTags := replaceSimilarTags(oldTags, similarTags)
				if len(newTags) == 0 || sliceEqual(oldTags, newTags) {
					continue
				}

				flog.Info("[bookmark] %s update tags from %v to %v", bookmark.ID, oldTags, newTags)

				_, err = ability.Invoke(ctx.Context(), hub.CapBookmark, "detach_tags", map[string]any{
					"id":   bookmark.ID,
					"tags": oldTags,
				})
				if err != nil {
					flog.Error(fmt.Errorf("detach bookmark %s tags error: %w", bookmark.ID, err))
					continue
				}

				_, err = ability.Invoke(ctx.Context(), hub.CapBookmark, "attach_tags", map[string]any{
					"id":   bookmark.ID,
					"tags": newTags,
				})
				if err != nil {
					flog.Error(fmt.Errorf("attach bookmark %s tags error: %w", bookmark.ID, err))
					continue
				}
			}

			return nil
		},
	},
}
