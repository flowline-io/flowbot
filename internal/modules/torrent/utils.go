package torrent

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/hekmon/transmissionrpc/v3"
)

func torrentClear(ctx context.Context) error {
	client, err := transmission.GetClient()
	if err != nil {
		return fmt.Errorf("clear failed, %w", err)
	}

	list, err := client.TorrentGetAll(ctx)
	if err != nil {
		return fmt.Errorf("clear failed, %w", err)
	}
	flog.Debug("[torrent] total %d torrents", len(list))

	ids := make([]int64, 0, len(list))
	for _, torrent := range list {
		if *torrent.Status == transmissionrpc.TorrentStatusSeed {
			ids = append(ids, *torrent.ID)
		}
	}
	if len(ids) > 0 {
		err = client.TorrentRemove(ctx, ids)
		if err != nil {
			return fmt.Errorf("clear failed, %w", err)
		}
		flog.Info("[torrent] cleared %d torrents", len(ids))
	}

	return nil
}
