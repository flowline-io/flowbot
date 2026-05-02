package hub

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "hub_health_check",
		Help:  "Periodic hub health check with alerting",
		Scope: cron.CronScopeSystem,
		When:  "*/5 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			checker := hub.NewChecker(hub.Default)
			result := checker.Check(context.Background())

			if result.Status != hub.HealthHealthy {
				msg := notify.Message{
					Title:    "Hub Health Alert",
					Body:     fmt.Sprintf("Hub status is %s at %s", result.Status, result.Timestamp.Format("15:04:05")),
					Priority: notify.High,
				}

				if err := notify.ChannelSend(types.Uid("system"), "hub_health", msg); err != nil {
					flog.Error(fmt.Errorf("hub health notify: %w", err))
				}
			}

			return nil
		},
	},
}
