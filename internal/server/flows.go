package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/flows"
	"github.com/flowline-io/flowbot/pkg/flog"
	"go.uber.org/fx"
)

// handleFlowQueue manages the flow execution queue lifecycle
func handleFlowQueue(lc fx.Lifecycle, queue *flows.QueueManager) {
	if queue == nil {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := queue.Start(ctx); err != nil {
				flog.Error(err)
				return err
			}
			flog.Info("flow queue manager started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := queue.Stop(ctx); err != nil {
				flog.Error(err)
				return err
			}
			flog.Info("flow queue manager stopped")
			return nil
		},
	})
}
