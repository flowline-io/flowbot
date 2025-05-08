package server

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"go.uber.org/fx"
)

func handlePlatform(lc fx.Lifecycle, driver protocol.Driver) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go driver.WebSocketClient()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return driver.Shoutdown()
		},
	})
}
