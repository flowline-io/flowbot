package reader

import (
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	rssClient "miniflux.app/v2/client"
)

var cronRules = []cron.Rule{
	{
		Name:  "reader_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(miniflux.ID, miniflux.EndpointKey)
			apiKey, _ := providers.GetConfig(miniflux.ID, miniflux.ApikeyKey)
			client := miniflux.NewMiniflux(endpoint.String(), apiKey.String())

			// total
			result, err := client.GetEntries(&rssClient.Filter{Limit: 1})
			if err != nil {
				flog.Error(err)
				return nil
			}
			stats.ReaderTotalCounter().Set(uint64(result.Total))

			// unread total
			result, err = client.GetEntries(&rssClient.Filter{Status: rssClient.EntryStatusUnread, Limit: 1})
			if err != nil {
				flog.Error(err)
				return nil
			}
			stats.ReaderUnreadTotalCounter().Set(uint64(result.Total))
			cache.SetInt64(stats.ReaderUnreadTotalStatsName, int64(result.Total))

			return nil
		},
	},
}
