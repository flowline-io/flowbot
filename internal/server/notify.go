package server

import (
	message_pusher "github.com/flowline-io/flowbot/pkg/notify/message-pusher"
	"github.com/flowline-io/flowbot/pkg/notify/ntfy"
	"github.com/flowline-io/flowbot/pkg/notify/pushover"
	"github.com/flowline-io/flowbot/pkg/notify/slack"
	"go.uber.org/fx"
)


var NotifyModules = fx.Options(
	fx.Invoke(
		message_pusher.Register,
		ntfy.Register,
		pushover.Register,
		slack.Register,
	),
)
