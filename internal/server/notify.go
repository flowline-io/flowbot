package server

import (
	"context"

	abilitynotify "github.com/flowline-io/flowbot/pkg/ability/notify"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	message_pusher "github.com/flowline-io/flowbot/pkg/notify/message-pusher"
	"github.com/flowline-io/flowbot/pkg/notify/ntfy"
	"github.com/flowline-io/flowbot/pkg/notify/pushover"
	"github.com/flowline-io/flowbot/pkg/notify/slack"
	"github.com/flowline-io/flowbot/pkg/rdb"

	"go.uber.org/fx"
)

var NotifyModules = fx.Options(
	fx.Invoke(
		message_pusher.Register,
		ntfy.Register,
		pushover.Register,
		slack.Register,
		initNotificationGateway,
	),
)

func initNotificationGateway(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// initialize template engine
			if err := notifytmpl.Init(); err != nil {
				return err
			}

			// initialize rule engine with Redis client
			if err := notifyrules.Init(rdb.Client); err != nil {
				return err
			}

			// register notify capability with ability framework
			return abilitynotify.Register()
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}
