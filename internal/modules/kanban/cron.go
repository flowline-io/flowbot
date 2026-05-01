package kanban

import (
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

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

			tasks, _ := res.Data.([]*ability.Task)
			taskTotal := len(tasks)

			stats.KanbanTaskTotalCounter().Set(uint64(taskTotal))
			rdb.SetMetricsInt64(stats.KanbanTaskTotalStatsName, int64(taskTotal))

			return nil
		},
	},
}
