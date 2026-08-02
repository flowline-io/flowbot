package store

import (
	"context"
	"maps"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

// WorkflowCatalogAdapter adapts WorkflowStore to workflow.Catalog (model DTOs).
type WorkflowCatalogAdapter struct {
	S *WorkflowStore
}

// NewWorkflowCatalogAdapter wraps a WorkflowStore as workflow.Catalog.
func NewWorkflowCatalogAdapter(s *WorkflowStore) workflow.Catalog {
	if s == nil {
		return nil
	}
	return WorkflowCatalogAdapter{S: s}
}

// GetMetadata implements workflow.DefinitionStore by loading gen rows and mapping to types.
func (a WorkflowCatalogAdapter) GetMetadata(ctx context.Context, name string) (*types.WorkflowMetadata, error) {
	dto, err := a.S.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if dto == nil {
		return nil, types.Errorf(types.ErrNotFound, "workflow %s", name)
	}
	return workflow.MetadataFromRows(workflowRowsFromGen(dto))
}

func workflowRowsFromGen(dto *WorkflowDefinitionDTO) workflow.WorkflowRows {
	if dto == nil {
		return workflow.WorkflowRows{}
	}
	tasks := make([]*model.WorkflowTaskRow, 0, len(dto.Tasks))
	for _, t := range dto.Tasks {
		if t == nil {
			continue
		}
		tasks = append(tasks, &model.WorkflowTaskRow{
			TaskID:   t.TaskID,
			Action:   t.Action,
			Describe: t.Describe,
			Params:   cloneJSONMap(t.Params),
			Vars:     append([]string(nil), t.Vars...),
			Conn:     append([]string(nil), t.Conn...),
			Retry:    cloneJSONMap(t.Retry),
		})
	}
	triggers := make([]*model.WorkflowTrigger, 0, len(dto.Triggers))
	for _, tr := range dto.Triggers {
		if tr == nil {
			continue
		}
		triggers = append(triggers, &model.WorkflowTrigger{
			ID:         tr.ID,
			WorkflowID: tr.WorkflowID,
			Type:       tr.Type,
			Enabled:    tr.Enabled,
			Rule:       cloneJSONMap(tr.Rule),
		})
	}
	return workflow.WorkflowRows{
		Workflow: mapWorkflowDTO(dto.Workflow),
		Tasks:    tasks,
		Triggers: triggers,
	}
}

func cloneJSONMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	maps.Copy(out, m)
	return out
}

// ApplyDefinition implements workflow.Catalog.
func (a WorkflowCatalogAdapter) ApplyDefinition(ctx context.Context, meta *types.WorkflowMetadata) (*model.Workflow, error) {
	row, err := a.S.ApplyDefinition(ctx, meta)
	if err != nil {
		return nil, err
	}
	return mapWorkflowDTO(row), nil
}

// ListDefinitions implements workflow.Catalog.
func (a WorkflowCatalogAdapter) ListDefinitions(ctx context.Context) ([]*model.Workflow, error) {
	rows, err := a.S.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	return mapWorkflowDTOs(rows), nil
}

// DeleteDefinitionByName implements workflow.Catalog.
func (a WorkflowCatalogAdapter) DeleteDefinitionByName(ctx context.Context, name string) error {
	return a.S.DeleteDefinitionByName(ctx, name)
}

// ListRunsByName implements workflow.Catalog.
func (a WorkflowCatalogAdapter) ListRunsByName(ctx context.Context, name string) ([]*model.WorkflowRun, error) {
	rows, err := a.S.ListRunsByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapWorkflowRunDTOs(rows), nil
}

// WorkflowRunStoreAdapter adapts WorkflowRunStore to workflow.WorkflowRunStore.
type WorkflowRunStoreAdapter struct {
	S *WorkflowRunStore
}

// NewWorkflowRunStoreAdapter wraps a WorkflowRunStore as workflow.WorkflowRunStore.
func NewWorkflowRunStoreAdapter(s *WorkflowRunStore) workflow.WorkflowRunStore {
	if s == nil {
		return nil
	}
	return WorkflowRunStoreAdapter{S: s}
}

// CreateRun implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) CreateRun(ctx context.Context, workflowID int64, workflowName, workflowFile, triggerType string, triggerInfo, inputParams map[string]any) (*model.WorkflowRun, error) {
	row, err := a.S.CreateRun(ctx, workflowID, workflowName, workflowFile, triggerType, triggerInfo, inputParams)
	if err != nil {
		return nil, err
	}
	return mapWorkflowRunDTO(row), nil
}

// UpdateRunStatus implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error {
	return a.S.UpdateRunStatus(ctx, runID, status, errMsg)
}

// CreateStepRun implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params map[string]any, attempt int) (*model.WorkflowStepRun, error) {
	row, err := a.S.CreateStepRun(ctx, runID, stepID, stepName, action, actionType, params, attempt)
	if err != nil {
		return nil, err
	}
	return mapWorkflowStepRunDTO(row), nil
}

// UpdateStepRun implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	return a.S.UpdateStepRun(ctx, stepRunID, status, result, errMsg, attempt)
}

// SaveCheckpoint implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	return a.S.SaveCheckpoint(ctx, runID, data)
}

// GetIncompleteRuns implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) GetIncompleteRuns(ctx context.Context) ([]*model.WorkflowRun, error) {
	rows, err := a.S.GetIncompleteRuns(ctx)
	if err != nil {
		return nil, err
	}
	return mapWorkflowRunDTOs(rows), nil
}

// GetCheckpoint implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	return a.S.GetCheckpoint(ctx, runID, target)
}

// GetRun implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error) {
	row, err := a.S.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return mapWorkflowRunDTO(row), nil
}

// UpdateRunHeartbeat implements workflow.WorkflowRunStore.
func (a WorkflowRunStoreAdapter) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	return a.S.UpdateRunHeartbeat(ctx, runID)
}

func mapWorkflowDTO(row *gen.Workflow) *model.Workflow {
	if row == nil {
		return nil
	}
	return &model.Workflow{
		ID:             row.ID,
		Name:           row.Name,
		Describe:       row.Describe,
		Enabled:        row.Enabled,
		Resumable:      row.Resumable,
		MaxConcurrency: row.MaxConcurrency,
		Pipeline:       append([]string(nil), row.Pipeline...),
		Inputs:         cloneJSONMapSlice(row.Inputs),
	}
}

func cloneJSONMapSlice(in []map[string]any) []map[string]any {
	if in == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		out = append(out, cloneJSONMap(m))
	}
	return out
}

func mapWorkflowDTOs(rows []*gen.Workflow) []*model.Workflow {
	out := make([]*model.Workflow, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflowDTO(row))
	}
	return out
}

func mapWorkflowRunDTO(row *gen.WorkflowRun) *model.WorkflowRun {
	if row == nil {
		return nil
	}
	return &model.WorkflowRun{
		ID:           row.ID,
		WorkflowID:   row.WorkflowID,
		WorkflowName: row.WorkflowName,
		Status:       row.Status,
		TriggerType:  row.TriggerType,
		StartedAt:    row.StartedAt,
		CreatedAt:    row.CreatedAt,
		CompletedAt:  row.CompletedAt,
		Error:        row.Error,
	}
}

func mapWorkflowRunDTOs(rows []*gen.WorkflowRun) []*model.WorkflowRun {
	out := make([]*model.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflowRunDTO(row))
	}
	return out
}

func mapWorkflowStepRunDTO(row *gen.WorkflowStepRun) *model.WorkflowStepRun {
	if row == nil {
		return nil
	}
	return &model.WorkflowStepRun{
		ID:          row.ID,
		StepID:      row.StepID,
		StepName:    row.StepName,
		Action:      row.Action,
		ActionType:  row.ActionType,
		Attempt:     row.Attempt,
		Status:      row.Status,
		StartedAt:   row.StartedAt,
		CompletedAt: row.CompletedAt,
		Error:       row.Error,
		Params:      row.Params,
		Result:      row.Result,
	}
}
