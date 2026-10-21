package store

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// PipelineRunStoreAdapter adapts PipelineStore to pipeline.RunStore (model DTOs).
type PipelineRunStoreAdapter struct {
	S *PipelineStore
}

// NewPipelineRunStoreAdapter wraps a PipelineStore as pipeline.RunStore.
func NewPipelineRunStoreAdapter(s *PipelineStore) pipeline.RunStore {
	if s == nil {
		return nil
	}
	return PipelineRunStoreAdapter{S: s}
}

// CreateRun implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (*model.PipelineRun, error) {
	row, err := a.S.CreateRun(ctx, pipelineName, eventID, eventType, triggerSource)
	if err != nil {
		return nil, err
	}
	return mapPipelineRunDTO(row), nil
}

// UpdateRunStatus implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error {
	return a.S.UpdateRunStatus(ctx, runID, status, errMsg)
}

// CreateStepRun implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) CreateStepRun(ctx context.Context, runID int64, stepName, capName, operation string, params map[string]any, attempt int) (*model.PipelineStepRun, error) {
	row, err := a.S.CreateStepRun(ctx, runID, stepName, capName, operation, params, attempt)
	if err != nil {
		return nil, err
	}
	return mapPipelineStepRunDTO(row), nil
}

// UpdateStepRun implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	return a.S.UpdateStepRun(ctx, stepRunID, status, result, errMsg, attempt)
}

// ListStepRunsByID implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) ListStepRunsByID(ctx context.Context, runID int64) ([]*model.PipelineStepRun, error) {
	rows, err := a.S.GetStepRunsByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	return mapPipelineStepRunDTOs(rows), nil
}

// ListStepRuns implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) ListStepRuns(ctx context.Context, runID int64) ([]*model.PipelineStepRun, error) {
	rows, err := a.S.ListStepRunsByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	return mapPipelineStepRunDTOs(rows), nil
}

// ClaimFailedRun implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) ClaimFailedRun(ctx context.Context, runID int64) error {
	return a.S.ClaimFailedRun(ctx, runID)
}

// PrepareStepRetry implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) PrepareStepRetry(ctx context.Context, stepRunID int64, params map[string]any, attempt int) error {
	return a.S.PrepareStepRetry(ctx, stepRunID, params, attempt)
}

// SaveCheckpoint implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	return a.S.SaveCheckpoint(ctx, runID, data)
}

// GetIncompleteRuns implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) GetIncompleteRuns(ctx context.Context) ([]*model.PipelineRun, error) {
	rows, err := a.S.GetIncompleteRuns(ctx)
	if err != nil {
		return nil, err
	}
	return mapPipelineRunDTOs(rows), nil
}

// GetCheckpoint implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	return a.S.GetCheckpoint(ctx, runID, target)
}

// GetRun implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error) {
	row, err := a.S.GetRun(ctx, runID)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return mapPipelineRunDTO(row), nil
}

// UpdateRunHeartbeat implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	return a.S.UpdateRunHeartbeat(ctx, runID)
}

// HasConsumed implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error) {
	return a.S.HasConsumed(ctx, consumerName, eventID)
}

// RecordConsumption implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) RecordConsumption(ctx context.Context, consumerName, eventID string) error {
	return a.S.RecordConsumption(ctx, consumerName, eventID)
}

// RecordResourceLink implements pipeline.RunStore.
func (a PipelineRunStoreAdapter) RecordResourceLink(ctx context.Context, link model.ResourceLink) error {
	return a.S.RecordResourceLink(ctx, &gen.ResourceLink{
		SourceEventID:    link.SourceEventID,
		TargetEventID:    link.TargetEventID,
		SourceApp:        link.SourceApp,
		TargetApp:        link.TargetApp,
		SourceCapability: link.SourceCapability,
		TargetCapability: link.TargetCapability,
		SourceEntityID:   link.SourceEntityID,
		TargetEntityID:   link.TargetEntityID,
		PipelineRunID:    link.PipelineRunID,
		PipelineName:     link.PipelineName,
	})
}

// PipelineCatalogAdapter adapts PipelineStore to pipeline.DefinitionCatalog.
type PipelineCatalogAdapter struct {
	S *PipelineStore
}

// NewPipelineCatalogAdapter wraps a PipelineStore as pipeline.DefinitionCatalog.
func NewPipelineCatalogAdapter(s *PipelineStore) pipeline.DefinitionCatalog {
	if s == nil {
		return nil
	}
	return PipelineCatalogAdapter{S: s}
}

// CreateDefinition implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) CreateDefinition(ctx context.Context, name, description, createdBy string) error {
	return a.S.CreateDefinition(ctx, name, description, createdBy)
}

// GetDefinitionByName implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) GetDefinitionByName(ctx context.Context, name string) (*model.PipelineDefinition, error) {
	row, err := a.S.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapPipelineDefinitionDTO(row), nil
}

// UpdateDefinitionDraft implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error) {
	row, err := a.S.UpdateDefinitionDraft(ctx, name, yamlDraft, version)
	if err != nil {
		return nil, err
	}
	return mapPipelineDefinitionDTO(row), nil
}

// PublishDefinition implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) PublishDefinition(ctx context.Context, name string, version int) (*model.PipelineDefinition, error) {
	row, err := a.S.PublishDefinition(ctx, name, version)
	if err != nil {
		return nil, err
	}
	return mapPipelineDefinitionDTO(row), nil
}

// EnsureDefinitionCreatedBy implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) EnsureDefinitionCreatedBy(ctx context.Context, name, createdBy string) error {
	return a.S.EnsureDefinitionCreatedBy(ctx, name, createdBy)
}

// DeleteDefinitionByName implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) DeleteDefinitionByName(ctx context.Context, name string) (int64, error) {
	return a.S.DeleteDefinitionByName(ctx, name)
}

// ListPublishedDefinitions implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) ListPublishedDefinitions(ctx context.Context) ([]pipeline.DefinitionRecord, error) {
	return a.S.ListPublishedDefinitions(ctx)
}

// SetBlueprintOrigin implements pipeline.BlueprintOriginStore.
func (a PipelineCatalogAdapter) SetBlueprintOrigin(ctx context.Context, name string, origin pipeline.BlueprintOrigin) error {
	return a.S.SetBlueprintOrigin(ctx, name, origin)
}

// ClearBlueprintOrigin implements pipeline.BlueprintOriginStore.
func (a PipelineCatalogAdapter) ClearBlueprintOrigin(ctx context.Context, name string) error {
	return a.S.ClearBlueprintOrigin(ctx, name)
}

// ListLinkedPipelineNames implements pipeline.BlueprintOriginStore.
func (a PipelineCatalogAdapter) ListLinkedPipelineNames(ctx context.Context, source, blueprintID string) ([]string, error) {
	return a.S.ListLinkedPipelineNames(ctx, source, blueprintID)
}

// ListDefinitions implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) ListDefinitions(ctx context.Context) ([]*model.PipelineDefinition, error) {
	rows, err := a.S.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.PipelineDefinition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineDefinitionDTO(row))
	}
	return out, nil
}

// RenameDefinition implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) RenameDefinition(ctx context.Context, oldName, newName string) (*model.PipelineDefinition, error) {
	row, err := a.S.RenameDefinition(ctx, oldName, newName)
	if err != nil {
		return nil, err
	}
	return mapPipelineDefinitionDTO(row), nil
}

// SetDefinitionEnabled implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) SetDefinitionEnabled(ctx context.Context, name string, enabled bool) (*model.PipelineDefinition, error) {
	row, err := a.S.SetDefinitionEnabled(ctx, name, enabled)
	if err != nil {
		return nil, err
	}
	return mapPipelineDefinitionDTO(row), nil
}

// ListDefinitionVersions implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) ListDefinitionVersions(ctx context.Context, name string) ([]model.PipelineDefinitionVersion, error) {
	rows, err := a.S.ListDefinitionVersions(ctx, name)
	if err != nil {
		return nil, err
	}
	out := make([]model.PipelineDefinitionVersion, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineDefinitionVersion(row))
	}
	return out, nil
}

// GetDefinitionVersion implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) GetDefinitionVersion(ctx context.Context, name string, version int) (*model.PipelineDefinitionVersion, error) {
	row, err := a.S.GetDefinitionVersion(ctx, name, version)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, types.ErrNotFound
	}
	dto := mapPipelineDefinitionVersion(row)
	return &dto, nil
}

func mapPipelineDefinitionVersion(row *gen.PipelineDefinitionVersion) model.PipelineDefinitionVersion {
	return model.PipelineDefinitionVersion{
		Version:   row.Version,
		YAML:      row.Yaml,
		CreatedAt: row.CreatedAt,
	}
}

// LatestRunStartedAtByParentNames implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) LatestRunStartedAtByParentNames(ctx context.Context, names []string) (map[string]time.Time, error) {
	return a.S.LatestRunStartedAtByParentNames(ctx, names)
}

// RunLatencyStatsByParentNames implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) RunLatencyStatsByParentNames(ctx context.Context, names []string, since time.Time) (map[string]types.RunLatencyStats, error) {
	return a.S.RunLatencyStatsByParentNames(ctx, names, since)
}

// Stats implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) Stats(ctx context.Context, name string, since, until time.Time, groupBy string) (*types.PipelineStats, error) {
	return a.S.PipelineStats(ctx, name, since, until, groupBy)
}

// GetRunsByParentName implements pipeline.DefinitionCatalog.
func (a PipelineCatalogAdapter) GetRunsByParentName(ctx context.Context, parentName string) ([]*model.PipelineRun, error) {
	rows, err := a.S.GetRunsByParentName(ctx, parentName)
	if err != nil {
		return nil, err
	}
	return mapPipelineRunDTOs(rows), nil
}

// PipelineEventLoader adapts EventStore to pipeline.EventLoader.
type PipelineEventLoader struct {
	S *EventStore
}

// NewPipelineService wires a pipeline service to ps and events.
// The returned catalog is the adapter the service uses.
func NewPipelineService(ps *PipelineStore, events *EventStore) (*pipeline.Service, PipelineCatalogAdapter) {
	catalog := PipelineCatalogAdapter{S: ps}
	svc := pipeline.NewService(catalog)
	svc.SetRunStore(NewPipelineRunStoreAdapter(ps))
	svc.SetEventLoader(NewPipelineEventLoader(events))
	return svc, catalog
}

// NewPipelineEventLoader wraps an EventStore as pipeline.EventLoader.
func NewPipelineEventLoader(s *EventStore) pipeline.EventLoader {
	if s == nil {
		return nil
	}
	return PipelineEventLoader{S: s}
}

// LoadDataEvent implements pipeline.EventLoader.
func (a PipelineEventLoader) LoadDataEvent(ctx context.Context, eventID string) (types.DataEvent, error) {
	row, err := a.S.GetDataEventByEventID(ctx, eventID)
	if err != nil {
		return types.DataEvent{}, err
	}
	if row == nil {
		return types.DataEvent{}, types.ErrNotFound
	}
	return DataEventFromRow(row), nil
}

func mapPipelineDefinitionDTO(row *gen.PipelineDefinition) *model.PipelineDefinition {
	if row == nil {
		return nil
	}
	return &model.PipelineDefinition{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		Status:          string(row.Status),
		Version:         row.Version,
		CreatedBy:       row.CreatedBy,
		YamlDraft:       row.YamlDraft,
		YamlPublished:   row.YamlPublished,
		BlueprintSource: row.BlueprintSource,
		BlueprintID:     row.BlueprintID,
		BlueprintHash:   row.BlueprintHash,
		BlueprintYAML:   row.BlueprintYaml,
		BlueprintInputs: row.BlueprintInputs,
	}
}

func mapPipelineRunDTO(row *gen.PipelineRun) *model.PipelineRun {
	if row == nil {
		return nil
	}
	return &model.PipelineRun{
		ID:            row.ID,
		PipelineName:  row.PipelineName,
		EventID:       row.EventID,
		EventType:     row.EventType,
		TriggerSource: string(row.TriggerSource),
		Status:        row.Status,
		Error:         row.Error,
		CreatedAt:     row.CreatedAt,
		StartedAt:     row.StartedAt,
		CompletedAt:   row.CompletedAt,
	}
}

func mapPipelineRunDTOs(rows []*gen.PipelineRun) []*model.PipelineRun {
	out := make([]*model.PipelineRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineRunDTO(row))
	}
	return out
}

func mapPipelineStepRunDTO(row *gen.PipelineStepRun) *model.PipelineStepRun {
	if row == nil {
		return nil
	}
	return &model.PipelineStepRun{
		ID:          row.ID,
		StepName:    row.StepName,
		Capability:  row.Capability,
		Operation:   row.Operation,
		Status:      row.Status,
		Attempt:     row.Attempt,
		StartedAt:   row.StartedAt,
		CompletedAt: row.CompletedAt,
		Error:       row.Error,
		Params:      row.Params,
		Result:      row.Result,
	}
}

func mapPipelineStepRunDTOs(rows []*gen.PipelineStepRun) []*model.PipelineStepRun {
	out := make([]*model.PipelineStepRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineStepRunDTO(row))
	}
	return out
}
