package kanban

import (
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cacheStore *cache.RedisStore

// SetCacheStore sets the cache store for kanban module cron.
func SetCacheStore(store *cache.RedisStore) {
	cacheStore = store
}

var cronRules = []cron.Rule{
	{
		Name:  "kanban_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanListTasks, map[string]any{})
			if err != nil {
				flog.Warn("%s", err.Error())
				return nil
			}

			tasks, ok := res.Data.([]*ability.Task)
			if !ok {
				tasks = nil
			}
			taskTotal := 0
			if tasks != nil {
				taskTotal = len(tasks)
			}

			stats.KanbanTaskTotalCounter().Set(uint64(taskTotal))
			cacheStore.SetMetricsInt64(stats.KanbanTaskTotalStatsName, int64(taskTotal))

			return nil
		},
	},
}
