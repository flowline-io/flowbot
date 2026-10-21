package workflow

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (s *Service) ready() error {
	if s == nil || s.catalog == nil {
		return types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	return nil
}

// ListTriggers returns every workflow trigger row.
func (s *Service) ListTriggers(ctx context.Context) ([]*model.WorkflowTrigger, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.ListTriggers(ctx)
}

// TriggersByName returns trigger rows for one workflow, including database ids.
func (s *Service) TriggersByName(ctx context.Context, name string) ([]*model.WorkflowTrigger, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.TriggersByName(ctx, name)
}

// SetEnabled updates the workflow enabled flag and reloads triggers.
// A reload failure is logged. The update still succeeds.
func (s *Service) SetEnabled(ctx context.Context, name string, enabled bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if _, err := s.catalog.SetEnabled(ctx, name, enabled); err != nil {
		return err
	}
	if reloadErr := s.ReloadTriggers(ctx); reloadErr != nil {
		flog.Warn("workflow: reload triggers after enabled toggle: %v", reloadErr)
	}
	return nil
}

// SetTriggerEnabled updates one trigger and reloads triggers.
// A reload failure is logged. The update still succeeds.
func (s *Service) SetTriggerEnabled(ctx context.Context, name string, triggerID int64, enabled bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if _, err := s.catalog.SetTriggerEnabled(ctx, name, triggerID, enabled); err != nil {
		return err
	}
	if reloadErr := s.ReloadTriggers(ctx); reloadErr != nil {
		flog.Warn("workflow: reload triggers after trigger enabled toggle: %v", reloadErr)
	}
	return nil
}

// LatestRunStartedAt returns the newest run start time for each workflow name.
func (s *Service) LatestRunStartedAt(ctx context.Context, names []string) (map[string]time.Time, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.LatestRunStartedAtByNames(ctx, names)
}

// RunLatencyStats returns recent run latency for each workflow name.
func (s *Service) RunLatencyStats(ctx context.Context, names []string, since time.Time) (map[string]types.RunLatencyStats, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.RunLatencyStatsByNames(ctx, names, since)
}

// Stats returns aggregated run statistics for one workflow, or every workflow when name is empty.
func (s *Service) Stats(ctx context.Context, name string, since, until time.Time, groupBy string) (*types.WorkflowStats, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.Stats(ctx, name, since, until, groupBy)
}

// GetRun returns one workflow run.
func (s *Service) GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error) {
	if s == nil || s.runs == nil {
		return nil, types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	run, err := s.runs.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, types.ErrNotFound
	}
	return run, nil
}

// ListStepRuns returns step runs for a workflow run.
func (s *Service) ListStepRuns(ctx context.Context, runID int64) ([]*model.WorkflowStepRun, error) {
	if s == nil || s.runs == nil {
		return nil, types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	return s.runs.ListStepRuns(ctx, runID)
}
