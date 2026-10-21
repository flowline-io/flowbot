package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Service orchestrates analytics queries across domain stores.
type Service struct{}

// NewService returns an analytics service.
func NewService() *Service {
	return &Service{}
}

// Overview returns Personal + Ops headline counters for the landing page.
func (*Service) Overview(ctx context.Context, uid string, since, until time.Time, activeRange, rangeLabel string) (*types.AnalyticsOverview, error) {
	out := &types.AnalyticsOverview{
		PeriodStart: since.UTC().Format("2006-01-02"),
		PeriodEnd:   until.UTC().Format("2006-01-02"),
		ActiveRange: activeRange,
		RangeLabel:  rangeLabel,
	}
	if err := fillPersonalOverview(ctx, out, uid, since, until); err != nil {
		return nil, err
	}
	if err := fillOpsOverview(ctx, out, since, until); err != nil {
		return nil, err
	}
	return out, nil
}

func fillPersonalOverview(ctx context.Context, out *types.AnalyticsOverview, uid string, since, until time.Time) error {
	usageStore := store.NewLLMUsageStoreFromDatabase()
	if usageStore != nil {
		tokenStats, err := usageStore.TokenUsageStats(ctx, uid, since, until, "model")
		if err != nil {
			return fmt.Errorf("analytics: token overview: %w", err)
		}
		out.TokenTotal = tokenStats.Summary.TotalTokens
	}

	eventStore := store.EventStoreFromDB()
	if eventStore != nil {
		eventStats, err := eventStore.EventStats(ctx, uid, since, until, "event_type")
		if err != nil {
			return fmt.Errorf("analytics: event overview: %w", err)
		}
		out.EventTotal = eventStats.Summary.TotalEvents
	}

	activity, err := loadActivityStats(ctx, uid, since, until)
	if err != nil {
		return err
	}
	out.PipelineRuns = activity.Summary.PipelineRuns
	out.FunctionRuns = activity.Summary.FunctionRuns
	return nil
}

func fillOpsOverview(ctx context.Context, out *types.AnalyticsOverview, since, until time.Time) error {
	pipe, wf, fn, err := loadOpsHealth(ctx, since, until, "day")
	if err != nil {
		return err
	}
	var success, failed int64
	if pipe != nil {
		success += pipe.Summary.SuccessfulRuns
		failed += pipe.Summary.FailedRuns
	}
	if wf != nil {
		success += wf.Summary.SuccessfulRuns
		failed += wf.Summary.FailedRuns
	}
	if fn != nil {
		success += fn.Summary.SuccessfulRuns
		failed += fn.Summary.FailedRuns
	}
	out.OpsFailedRuns = failed
	total := success + failed
	if total > 0 {
		out.OpsSuccessRate = float64(success) / float64(total) * 100
	}
	return nil
}

func loadOpsHealth(ctx context.Context, since, until time.Time, groupBy string) (
	pipe *types.PipelineStats,
	wf *types.WorkflowStats,
	fn *types.FunctionStats,
	err error,
) {
	if groupBy == "" {
		groupBy = "day"
	}
	if ps := store.PipelineStoreFromDB(); ps != nil {
		pipe, err = ps.PipelineStats(ctx, "", since, until, groupBy)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("analytics: pipeline health: %w", err)
		}
	}
	if ws := store.WorkflowStoreFromDB(); ws != nil {
		wf, err = ws.WorkflowStats(ctx, "", since, until, groupBy)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("analytics: workflow health: %w", err)
		}
	}
	if fs := store.FunctionStoreFromDB(); fs != nil {
		fn, err = fs.FunctionStats(ctx, "", since, until, groupBy)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("analytics: function health: %w", err)
		}
	}
	return pipe, wf, fn, nil
}

func loadActivityStats(ctx context.Context, uid string, since, until time.Time) (*types.ActivityStats, error) {
	out := &types.ActivityStats{
		Summary:     types.ActivityStatsSummary{},
		Series:      []types.ActivityStatsSeries{},
		PeriodStart: since.UTC().Format("2006-01-02"),
		PeriodEnd:   until.UTC().Format("2006-01-02"),
	}
	ps := store.PipelineStoreFromDB()
	if ps != nil {
		pipeSeries, pipeTotal, err := ps.OwnedRunActivity(ctx, uid, since, until)
		if err != nil {
			return nil, fmt.Errorf("analytics: pipeline activity: %w", err)
		}
		out.Summary.PipelineRuns = pipeTotal
		out.Series = append(out.Series, types.ActivityStatsSeries{
			Kind: "pipeline", Label: "pipeline", Points: pipeSeries,
		})
	}
	fs := store.FunctionStoreFromDB()
	if fs != nil {
		fnSeries, fnTotal, err := fs.OwnedRunActivity(ctx, uid, since, until)
		if err != nil {
			return nil, fmt.Errorf("analytics: function activity: %w", err)
		}
		out.Summary.FunctionRuns = fnTotal
		out.Series = append(out.Series, types.ActivityStatsSeries{
			Kind: "function", Label: "function", Points: fnSeries,
		})
	}
	return out, nil
}

// TokenUsage proxies LLM usage stats for the signed-in user.
func (*Service) TokenUsage(ctx context.Context, uid string, since, until time.Time, groupBy string) (*types.TokenUsageStats, error) {
	usageStore := store.NewLLMUsageStoreFromDatabase()
	if usageStore == nil {
		return &types.TokenUsageStats{}, nil
	}
	return usageStore.TokenUsageStats(ctx, uid, since, until, groupBy)
}

// EventStats proxies data_events aggregation for the signed-in user.
func (*Service) EventStats(ctx context.Context, uid string, since, until time.Time, groupBy string) (*types.EventStats, error) {
	eventStore := store.EventStoreFromDB()
	if eventStore == nil {
		return &types.EventStats{}, nil
	}
	return eventStore.EventStats(ctx, uid, since, until, groupBy)
}

// ActivityStats returns personal pipeline/function run counts.
func (*Service) ActivityStats(ctx context.Context, uid string, since, until time.Time) (*types.ActivityStats, error) {
	return loadActivityStats(ctx, uid, since, until)
}

// OpsRunsHealth returns site-wide pipeline, workflow, and function health stats.
func (*Service) OpsRunsHealth(ctx context.Context, since, until time.Time, groupBy string) (
	pipe *types.PipelineStats,
	wf *types.WorkflowStats,
	fn *types.FunctionStats,
	err error,
) {
	return loadOpsHealth(ctx, since, until, groupBy)
}
