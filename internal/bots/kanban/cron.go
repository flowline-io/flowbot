package kanban

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
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
			client, err := kanboard.GetClient()
			list, err := client.GetAllTasks(ctx.Context(), kanboard.DefaultProjectId, kanboard.Active)
			if err != nil {
				flog.Error(err)
				return nil
			}

			taskTotal := len(list)

			stats.KanbanTaskTotalCounter().Set(uint64(taskTotal))
			rdb.SetInt64(stats.KanbanTaskTotalStatsName, int64(taskTotal))

			return nil
		},
	},
}
