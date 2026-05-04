package bookmark

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.BookmarkArchiveBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			id, _ := param.String("id")

			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, ability.OpBookmarkArchive, map[string]any{"id": id})
			if err != nil {
				return fmt.Errorf("failed to archive bookmark: %w", err)
			}

			err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "bookmark.archived", []string{"slack", "ntfy"}, map[string]any{
				"title": res.Text,
				"id":    id,
			})
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			return nil
		},
	},
	{
		Id: types.BookmarkCreateBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			url, _ := param.String("url")

			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, ability.OpBookmarkCreate, map[string]any{"url": url})
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "bookmark.created", []string{"slack", "ntfy"}, map[string]any{
				"title": res.Text,
				"url":   url,
			})
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			return nil
		},
	},
	{
		Id: types.ArchiveBoxAddBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			url, _ := param.String("url")

			res, err := ability.Invoke(ctx.Context(), hub.CapArchive, ability.OpArchiveAdd, map[string]any{"url": url})
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "archive.item.added", []string{"slack", "ntfy"}, map[string]any{
				"title": res.Text,
				"url":   url,
			})
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			return nil
		},
	},
}
