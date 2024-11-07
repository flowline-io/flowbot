package dev

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/flog"
)

var cronRules = []cron.Rule{
	{
		Name: "dev_demo",
		Help: "cron example",
		When: "0 */1 * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
	{
		Name: "docker_images_prune",
		Help: "Docker images prune",
		When: "0 4 * * *",
		Action: func(types.Context) []types.MsgPayload {
			dc, err := client.NewClientWithOpts(client.FromEnv)
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
