package server

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/pkg/flog"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/redis/go-redis/v9"
)

var cronRules = []cron.Rule{
	{
		Name:  "server_user_online_change",
		Scope: cron.CronScopeUser,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			ctx_ := context.Background()
			keys, _ := cache.DB.Keys(ctx_, "online:*").Result()

			currentCount := int64(len(keys))
			lastKey := fmt.Sprintf("server:cron:online_count_last:%s", ctx.AsUser.String())

			lastCount, _ := cache.DB.Get(ctx_, lastKey).Int64()
			cache.DB.Set(ctx_, lastKey, currentCount, redis.KeepTTL)

			if lastCount != currentCount {
				return nil
			}
			return nil
		},
	},
	{
		Name:  "docker_images_prune",
		Help:  "Docker images prune",
		Scope: cron.CronScopeSystem,
		When:  "0 4 * * *",
		Action: func(types.Context) []types.MsgPayload {
			dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				flog.Error(err)
				return nil
			}
			report, err := dc.ImagesPrune(context.Background(), filters.Args{})
			if err != nil {
				flog.Error(err)
				return nil
			}
			flog.Info("docker prune report: %+v", report)

			return nil
		},
	},
}
