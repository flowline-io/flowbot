package server

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/uptimekuma"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/redis/go-redis/v9"
	"runtime"
)

var cronRules = []cron.Rule{
	{
		Name:  "server_user_online_change",
		Scope: cron.CronScopeUser,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			keys, _ := rdb.Client.Keys(ctx.Context(), "online:*").Result()

			currentCount := int64(len(keys))
			lastKey := fmt.Sprintf("server:cron:online_count_last:%s", ctx.AsUser.String())

			lastCount, _ := rdb.Client.Get(ctx.Context(), lastKey).Int64()
			rdb.Client.Set(ctx.Context(), lastKey, currentCount, redis.KeepTTL)

			if lastCount != currentCount {
				return nil
			}
			return nil
		},
	},
	{
		Name:  "docker_images_prune",
		Scope: cron.CronScopeSystem,
		When:  "0 4 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if runtime.GOOS == "windows" {
				return nil
			}

			dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				flog.Error(err)
				return nil
			}
			report, err := dc.ImagesPrune(ctx.Context(), filters.Args{})
			if err != nil {
				flog.Error(err)
				return nil
			}
			flog.Info("docker prune report: %+v", report)

			return nil
		},
	},
	{
		Name:  "docker_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if runtime.GOOS == "windows" {
				return nil
			}

			dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				flog.Error(err)
				return nil
			}
			list, err := dc.ContainerList(ctx.Context(), container.ListOptions{All: true})
			if err != nil {
				flog.Error(err)
				return nil
			}

			total := int64(0)
			for _, item := range list {
				if _, ok := item.Labels["homepage.name"]; !ok {
					continue
				}
				total++
			}

			rdb.SetMetricsInt64(stats.DockerContainerTotalStatsName, total)
			stats.DockerContainerTotalCounter().Set(uint64(total))

			return nil
		},
	},
	{
		Name:  "monitor_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			client := uptimekuma.GetClient()
			metricFamilies, err := client.Metrics()
			if err != nil {
				flog.Error(fmt.Errorf("cron failed to get metrics: %w", err))
				return nil
			}

			var up, down int64
			for _, metricFamily := range metricFamilies {
				for _, metric := range metricFamily.GetMetric() {
					if metricFamily.GetName() == uptimekuma.MonitorStatusMetric {
						if metric.GetGauge().GetValue() == uptimekuma.UP {
							up++
						}
						if metric.GetGauge().GetValue() == uptimekuma.DOWN {
							down++
						}
					}
				}
			}
			rdb.SetMetricsInt64(stats.MonitorUpTotalStatsName, up)
			rdb.SetMetricsInt64(stats.MonitorDownTotalStatsName, down)

			return nil
		},
	},
	{
		Name:  "rules_updater",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			//err := rules.Updater(ctx.Context())
			//if err != nil {
			//	flog.Error(err)
			//}
			return nil
		},
	},
}
