package server

import (
	"context"
	"fmt"

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
}
