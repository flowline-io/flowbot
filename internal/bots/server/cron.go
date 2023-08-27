package server

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/cache"
)

var cronRules = []cron.Rule{
	{
		Name: "server_user_online_change",
		When: "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			ctx_ := context.Background()
			keys, _ := cache.DB.Keys(ctx_, "online:*").Result()

			currentCount := int64(len(keys))
			lastKey := fmt.Sprintf("server:cron:online_count_last:%s", ctx.AsUser.UserId())

			lastCount, _ := cache.DB.Get(ctx_, lastKey).Int64()
			cache.DB.Set(ctx_, lastKey, currentCount, redis.KeepTTL)

			if lastCount != currentCount {
				return nil
			}
			return nil
		},
	},
}
