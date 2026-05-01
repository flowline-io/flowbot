package bookmark

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"

	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.BookmarkArchiveBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			id, _ := param.String("id")

			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "archive", map[string]any{"id": id})
			if err != nil {
				return fmt.Errorf("failed to archive bookmark: %w", err)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("Bookmark archived: %s", res.Text),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
	{
		Id: types.BookmarkCreateBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			url, _ := param.String("url")
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, "create", map[string]any{"url": url})
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("Bookmark created: %s", res.Text),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
	{
		Id: types.ArchiveBoxAddBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			url, _ := param.String("url")
			res, err := ability.Invoke(ctx.Context(), hub.CapArchive, "add", map[string]any{"url": url})
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("ArchiveBox: Success - %s", res.Text),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
}
