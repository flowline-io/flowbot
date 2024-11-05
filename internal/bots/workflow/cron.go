package workflow

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/flog"
)

var cronRules = []cron.Rule{
	{
		Name: "clear_workflow_jobs",
		Help: "clear workflow jobs",
		When: "0 0 * * *",
		Action: func(types.Context) []types.MsgPayload {
			list, err := store.Database.ListJobsByFilter(types.JobFilter{EndedAt: time.Now().Add(-7 * 24 * time.Hour)})
			if err != nil {
				flog.Error(err)
				return nil
			}
			jobIds := make([]int64, 0, len(list))
			for _, item := range list {
				jobIds = append(jobIds, item.ID)
			}
			err = store.Database.DeleteJobByIds(jobIds)
			if err != nil {
				flog.Error(err)
				return nil
			}
			err = store.Database.DeleteStepByJobIds(jobIds)
			if err != nil {
				flog.Error(err)
				return nil
			}

			if len(list) > 0 {
				return []types.MsgPayload{
					types.TextMsg{Text: fmt.Sprintf("clear workflow jobs total: %d", len(list))},
				}
			}

			return nil
		},
	},
}
