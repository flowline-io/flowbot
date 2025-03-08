package torrent

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "torrent_clear",
		Scope: cron.CronScopeSystem,
		When:  "*/5 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			err := torrentClear(ctx.Context())
			if err != nil {
				flog.Error(err)
			}
			return nil
		},
	},
	{
		Name:  "torrent_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			client, err := transmission.GetClient()
			if err != nil {
				flog.Error(fmt.Errorf("torrent metrics failed, %w", err))
				return nil
			}

			list, err := client.TorrentGetAll(context.Background())
			if err != nil {
				flog.Error(fmt.Errorf("torrent metrics get all failed, %w", err))
				return nil
			}

			statusMap := make(map[string]uint64, 10)
			for _, torrent := range list {
				if torrent.Status == nil {
					continue
				}
				statusMap[torrent.Status.String()]++
			}
			for status, amount := range statusMap {
				stats.TorrentStatusTotalCounter(status).Set(amount)
			}
			stats.TorrentDownloadTotalCounter().Set(uint64(len(list)))
			cache.SetInt64(stats.TorrentDownloadTotalStatsName, int64(len(list)))

			return nil
		},
	},
}
