package server

import (
	"context"

	"github.com/bytedance/sonic"

	storeDB "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	abilitynotify "github.com/flowline-io/flowbot/pkg/capability/notify"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify/messagepusher"
	"github.com/flowline-io/flowbot/pkg/notify/ntfy"
	"github.com/flowline-io/flowbot/pkg/notify/pushover"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	"github.com/flowline-io/flowbot/pkg/notify/slack"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"

	"go.uber.org/fx"
)

var NotifyModules = fx.Options(
	fx.Invoke(
		messagepusher.Register,
		ntfy.Register,
		pushover.Register,
		slack.Register,
		initNotificationGateway,
	),
)

func initNotificationGateway(lc fx.Lifecycle, store *cache.RedisStore) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := notifytmpl.Init(); err != nil {
				return err
			}

			engine := notifyrules.GetEngine()
			if engine == nil {
				engine = notifyrules.New(store)
			}

			enabled := true
			rules, err := storeDB.Database.ListNotifyRules(ctx, storeDB.ListNotifyRuleOptions{Enabled: &enabled})
			if err != nil {
				flog.Warn("failed to load notify rules from DB: %v", err)
			} else {
				configRules := make([]config.NotifyRule, 0, len(rules))
				for _, r := range rules {
					if !r.Enabled {
						continue
					}
					var cond string
					if r.Condition != "" {
						cond = r.Condition
					}
					var params config.NotifyRuleParams
					if r.ParamsJSON != "" {
						if err := sonic.Unmarshal([]byte(r.ParamsJSON), &params); err != nil {
							flog.Warn("skipping notify rule %s: invalid params JSON: %v", r.RuleID, err)
							continue
						}
					}
					configRules = append(configRules, config.NotifyRule{
						ID:     r.RuleID,
						Action: config.NotifyRuleAction(r.Action),
						Match: config.NotifyRuleMatch{
							Event:   r.EventPattern,
							Channel: r.ChannelPattern,
						},
						Condition: cond,
						Priority:  r.Priority,
						Params:    params,
					})
				}
				if err := engine.LoadConfig(configRules); err != nil {
					return err
				}
			}

			return abilitynotify.Register()
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})
}
