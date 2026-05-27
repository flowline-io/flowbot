package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/providers/slack"
)

// OAuthModules registers all OAuth provider factories via fx.
var OAuthModules = fx.Options(
	fx.Invoke(
		github.Register,
		slack.Register,
		dropbox.Register,
	),
)
