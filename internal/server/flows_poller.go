package server

import (
	"context"

	"github.com/flowline-io/flowbot/internal/flows"
	"github.com/flowline-io/flowbot/pkg/flog"
	"go.uber.org/fx"
)

// handleFlowPoller manages the flow poller lifecycle.
func handleFlowPoller(lc fx.Lifecycle, poller *flows.Poller) {
	if poller == nil {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			poller.Start(ctx)
			flog.Info("flow poller started")
			return nil
		},
	})
}
