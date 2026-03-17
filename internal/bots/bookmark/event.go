package bookmark

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/archivebox"

	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.BookmarkArchiveBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			client := karakeep.GetClient()

			id, _ := param.String("id")

			bookmark, err := client.GetBookmark(id)
			if err != nil {
				return fmt.Errorf("failed to get bookmark %w", err)
			}

			_, err = client.ArchiveBookmark(id)
			if err != nil {
				return fmt.Errorf("failed to archive bookmark: %w", err)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("Bookmark archived: [%s](%s)", bookmark.GetTitle(), id),
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
			client := karakeep.GetClient()

			url, _ := param.String("url")
			item, err := client.CreateBookmark(url)
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("Bookmark created: %s", item.Id),
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
			client := archivebox.GetClient()

			url, _ := param.String("url")
			resp, err := client.Add(archivebox.Data{
				Urls:   []string{url},
				Parser: "auto",
			})
			if err != nil {
				flog.Error(err)
				return nil
			}

			status := "Success"
			if !resp.Success {
				status = "Failed"
				flog.Warn("[archivebox] add %s failed, result: %v, errors: %v, stdout: %s, stderr: %s",
					url, resp.Result, resp.Errors, resp.Stdout, resp.Stderr)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("ArchiveBox: %s - %s", status, url),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
}
