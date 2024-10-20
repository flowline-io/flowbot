package torrent

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/hekmon/transmissionrpc/v3"
)

func torrentClear(ctx context.Context) error {
	endpoint, _ := providers.GetConfig(transmission.ID, transmission.EndpointKey)
	c, err := transmission.NewTransmission(endpoint.String())
	if err != nil {
		return fmt.Errorf("clear failed, %w", err)
	}

	list, err := c.TorrentGetAll(ctx)
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
		err = c.TorrentRemove(ctx, ids)
		if err != nil {
			return fmt.Errorf("clear failed, %w", err)
		}
		flog.Info("[torrent] cleared %d torrents", len(ids))
	}

	return nil
}
