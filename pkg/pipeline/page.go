package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (s *Service) ready() error {
	if s == nil || s.catalog == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return nil
}

// Create stores an empty pipeline definition.
func (s *Service) Create(ctx context.Context, name, description, createdBy string) error {
	if err := s.ready(); err != nil {
		return err
	}
	return s.catalog.CreateDefinition(ctx, name, description, createdBy)
}

// GetDefinition returns the stored definition, including the draft YAML.
func (s *Service) GetDefinition(ctx context.Context, name string) (*model.PipelineDefinition, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.GetDefinitionByName(ctx, strings.TrimSpace(name))
}

// ListDefinitions returns every pipeline definition, drafts included.
func (s *Service) ListDefinitions(ctx context.Context) ([]*model.PipelineDefinition, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.ListDefinitions(ctx)
}

// SaveDraft writes yaml_draft when version still matches.
func (s *Service) SaveDraft(ctx context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.UpdateDefinitionDraft(ctx, name, yamlDraft, version)
}

// Publish checks the draft YAML, backfills created_by, publishes, and reloads the engine.
// A reload failure is logged. The publish still succeeds.
func (s *Service) Publish(ctx context.Context, name string, version int, createdBy string) (*model.PipelineDefinition, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	def, err := s.catalog.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if _, err := ParseEditorYAML(def.YamlDraft); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid pipeline YAML", err)
	}
	if err := s.catalog.EnsureDefinitionCreatedBy(ctx, name, createdBy); err != nil {
		flog.Error(fmt.Errorf("ensure pipeline created_by before publish: %w", err))
	}
	published, err := s.catalog.PublishDefinition(ctx, name, version)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(fmt.Errorf("reload pipeline engine after publish: %w", reloadErr))
	}
	return published, nil
}

// Rename renames a pipeline and reloads the engine.
// A reload failure is logged. The rename still succeeds.
func (s *Service) Rename(ctx context.Context, oldName, newName string) (*model.PipelineDefinition, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	def, err := s.catalog.RenameDefinition(ctx, oldName, newName)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(fmt.Errorf("reload pipeline engine after rename: %w", reloadErr))
	}
	return def, nil
}

// SetEnabled toggles the published YAML enabled flag and reloads the engine.
// A reload failure fails the call.
func (s *Service) SetEnabled(ctx context.Context, name string, enabled bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if _, err := s.catalog.SetDefinitionEnabled(ctx, name, enabled); err != nil {
		return err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		return types.WrapError(types.ErrInternal, "reload pipeline engine after enabled toggle", reloadErr)
	}
	return nil
}

// ListVersions returns published YAML snapshots, newest first.
func (s *Service) ListVersions(ctx context.Context, name string) ([]model.PipelineDefinitionVersion, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.ListDefinitionVersions(ctx, name)
}

// GetVersion returns one published YAML snapshot.
func (s *Service) GetVersion(ctx context.Context, name string, version int) (*model.PipelineDefinitionVersion, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.GetDefinitionVersion(ctx, name, version)
}

// LatestRunStartedAt returns the newest run start time for each parent pipeline name.
func (s *Service) LatestRunStartedAt(ctx context.Context, names []string) (map[string]time.Time, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.LatestRunStartedAtByParentNames(ctx, names)
}

// RunLatencyStats returns recent run latency for each parent pipeline name.
func (s *Service) RunLatencyStats(ctx context.Context, names []string, since time.Time) (map[string]types.RunLatencyStats, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.RunLatencyStatsByParentNames(ctx, names, since)
}

// Stats returns aggregated run statistics for one pipeline, or every pipeline when name is empty.
func (s *Service) Stats(ctx context.Context, name string, since, until time.Time, groupBy string) (*types.PipelineStats, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.catalog.Stats(ctx, name, since, until, groupBy)
}

// GetRun returns one pipeline run.
func (s *Service) GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error) {
	if s == nil || s.runs == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
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

// ListStepRuns returns step runs ordered by creation time.
func (s *Service) ListStepRuns(ctx context.Context, runID int64) ([]*model.PipelineStepRun, error) {
	if s == nil || s.runs == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return s.runs.ListStepRuns(ctx, runID)
}

// ListStepRunsByID returns step runs ordered by id, at most 200, for the run-detail page.
func (s *Service) ListStepRunsByID(ctx context.Context, runID int64) ([]*model.PipelineStepRun, error) {
	if s == nil || s.runs == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return s.runs.ListStepRunsByID(ctx, runID)
}

// Retry resumes a failed run that belongs to parent.
func (s *Service) Retry(ctx context.Context, parent string, runID int64) error {
	if s == nil || s.runs == nil || s.events == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	run, err := s.runs.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run == nil || !RunBelongsToParent(run.PipelineName, parent) {
		return types.ErrNotFound
	}
	if run.Status != int(types.PipelineFailed) {
		return types.WrapError(types.ErrInvalidArgument, ErrRetryNotFailed.Error(), ErrRetryNotFailed)
	}
	event, err := s.events.LoadDataEvent(ctx, run.EventID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ErrRetryEventMissing
		}
		return err
	}
	if strings.TrimSpace(event.EventID) == "" {
		return ErrRetryEventMissing
	}
	engine := ActiveEngine()
	if engine == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline engine not ready")
	}
	return engine.RetryFailedRun(ctx, runID, event)
}
