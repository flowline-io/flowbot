package metrics

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func Module() fx.Option {
	return fx.Module("metrics",
		fx.Provide(
			stats.NewStats,
			NewPipelineCollector,
			NewWorkflowCollector,
			NewEventCollector,
			NewCapabilityCollector,
			NewAgentCollector,
		),
	)
}
