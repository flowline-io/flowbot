package server

import (
	"context"

	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	agentdcg "github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func initChatAgentScheduler(lc fx.Lifecycle) {
	sched := chatagent.NewTaskScheduler()
	chatagent.SetGlobalScheduler(sched)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			agentdcg.Init()
			chatagent.StartSessionSummaryWorker(ctx)
			if err := sched.Start(ctx); err != nil {
				flog.Error(err)
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return sched.Stop(ctx)
		},
	})
}
