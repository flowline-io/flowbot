package server

import (
	"context"

	abilitynotify "github.com/flowline-io/flowbot/pkg/ability/notify"
	"github.com/flowline-io/flowbot/pkg/cache"
	message_pusher "github.com/flowline-io/flowbot/pkg/notify/message-pusher"
	"github.com/flowline-io/flowbot/pkg/notify/ntfy"
	"github.com/flowline-io/flowbot/pkg/notify/pushover"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	"github.com/flowline-io/flowbot/pkg/notify/slack"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"

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

func initNotificationGateway(lc fx.Lifecycle, store *cache.RedisStore) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// initialize template engine
			if err := notifytmpl.Init(); err != nil {
				return err
			}

			// initialize rule engine with RedisStore
			if err := notifyrules.Init(store); err != nil {
				return err
			}

			// register notify capability with ability framework
			return abilitynotify.Register()
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})
}
