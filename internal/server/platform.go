package server

import (
	"context"

	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func handlePlatform(lc fx.Lifecycle, driver protocol.Driver) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go driver.WebSocketClient()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return driver.Shutdown()
		},
	})
}
