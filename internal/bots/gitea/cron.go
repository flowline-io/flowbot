package gitea

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "gitea_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(types.Context) []types.MsgPayload {
			client, err := gitea.GetClient()
			if err != nil {
				flog.Error(fmt.Errorf("failed to create gitea client: %w", err))
				return nil
			}

			user, err := client.GetMyUserInfo()
			if err != nil {
				flog.Error(fmt.Errorf("failed to get user info: %w", err))
				return nil
			}

			issues, err := client.ListIssues(user.UserName, 1, 100)
			if err != nil {
				flog.Error(fmt.Errorf("failed to list issues: %w", err))
				return nil
			}
			stats.GiteaIssueTotalCounter("open").Set(uint64(len(issues)))
			rdb.SetInt64(stats.GiteaIssueTotalStatsName, int64(len(issues)))

			return nil
		},
	},
}
