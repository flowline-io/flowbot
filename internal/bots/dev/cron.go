package dev

import (
	"context"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "dev_demo",
		Help:  "cron example",
		Scope: cron.CronScopeSystem,
		When:  "0 */10 * * *",
		Action: func(types.Context) []types.MsgPayload {
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
