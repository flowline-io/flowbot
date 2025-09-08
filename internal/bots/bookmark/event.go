package bookmark

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/archivebox"

	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.BookmarkArchiveBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			client := hoarder.GetClient()

			id, _ := param.String("id")
			ok, err := client.ArchiveBookmark(id)
			if err != nil {
				return fmt.Errorf("failed to archive bookmark %w", err)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("bookmark %s archive %v", id, ok),
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
			client := hoarder.GetClient()

			url, _ := param.String("url")
			item, err := client.CreateBookmark(url)
			if err != nil {
				flog.Error(err)
				return nil
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("bookmark %s created", item.Id),
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

			status := "success"
			if !resp.Success {
				status = "failed"
				flog.Warn("[archivebox] add %s failed, result: %v, errors: %v, stdout: %s, stderr: %s",
					url, resp.Result, resp.Errors, resp.Stdout, resp.Stderr)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("archivebox add %v, %v", url, status),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
}
