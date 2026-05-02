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

			switch result.Status {
			case hub.HealthHealthy:
			case hub.HealthDegraded, hub.HealthUnhealthy:
				msg := notify.Message{
					Title: "Hub Health Alert",
					Body:  fmt.Sprintf("Hub status is %s at %s", result.Status, result.Timestamp.Format("15:04:05")),
				}

				if err := notify.Send(fmt.Sprintf("slack://%s/%s", "channel", "hub"), msg); err != nil {
					flog.Error(fmt.Errorf("hub health notify: %w", err))
				}
			}

			return nil
		},
	},
}
