package store

import (
	"context"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowsteprun"
	"github.com/flowline-io/flowbot/internal/store/model"
)

// WorkflowRunStore persists workflow runs, step runs, and checkpoint data.
type WorkflowRunStore struct {
	client *gen.Client
}

// NewWorkflowRunStore creates a WorkflowRunStore backed by the given ent client.
func NewWorkflowRunStore(client *gen.Client) *WorkflowRunStore {
	return &WorkflowRunStore{client: client}
}

// CreateRun inserts a new workflow run record.
func (s *WorkflowRunStore) CreateRun(ctx context.Context, workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	wr, err := s.client.WorkflowRun.Create().
		SetWorkflowName(workflowName).
		SetWorkflowFile(workflowFile).
		SetStatus(int(model.WorkflowRunRunning)).
		SetTriggerType(triggerType).
		SetTriggerInfo(map[string]any(triggerInfo)).
		SetInputParams(map[string]any(inputParams)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowRunToModel(wr), nil
}

// UpdateRunStatus updates the status, error, and completed_at of a workflow run.
func (s *WorkflowRunStore) UpdateRunStatus(ctx context.Context, runID int64, status model.WorkflowRunState, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	u := s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetStatus(int(status)).
		SetCompletedAt(now)
	if errMsg != "" {
		u = u.SetError(errMsg)
	}
	return u.Exec(ctx)
}

// CreateStepRun inserts a new workflow step run record.
func (s *WorkflowRunStore) CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	sr, err := s.client.WorkflowStepRun.Create().
		SetWorkflowRunID(runID).
		SetStepID(stepID).
		SetStepName(stepName).
		SetAction(action).
		SetActionType(actionType).
		SetParams(map[string]any(params)).
		SetAttempt(attempt).
		SetStatus(int(model.WorkflowRunRunning)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowStepRunToModel(sr), nil
}

// UpdateStepRun updates the status, result, error, and attempt count of a workflow step run.
// completed_at is only set for terminal states (Done, Failed).
func (s *WorkflowRunStore) UpdateStepRun(ctx context.Context, stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	u := s.client.WorkflowStepRun.Update().
		Where(workflowsteprun.IDEQ(stepRunID)).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == model.WorkflowRunDone || status == model.WorkflowRunFailed {
		u = u.SetCompletedAt(time.Now())
	}
	if result != nil {
		u = u.SetResult(map[string]any(result))
	}
	if errMsg != "" {
		u = u.SetError(errMsg)
	}
	return u.Exec(ctx)
}

// SaveCheckpoint persists the intermediate workflow run state.
func (s *WorkflowRunStore) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	cp := model.JSON{}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	if err := cp.Scan(raw); err != nil {
		return err
	}
	return s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetCheckpointData(map[string]any(cp)).
		Exec(ctx)
}

// GetIncompleteRuns returns workflow runs that are still running and may need recovery.
func (s *WorkflowRunStore) GetIncompleteRuns(ctx context.Context) ([]*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.WorkflowRun.Query().
		Where(workflowrun.StatusEQ(int(model.WorkflowRunRunning))).
		Order(gen.Asc(workflowrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*model.WorkflowRun, len(runs))
	for i, r := range runs {
		result[i] = genWorkflowRunToModel(r)
	}
	return result, nil
}

// GetCheckpoint loads the checkpoint data for a workflow run.
func (s *WorkflowRunStore) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
	wr, err := s.client.WorkflowRun.Query().
		Where(workflowrun.IDEQ(runID)).
		Select(workflowrun.FieldCheckpointData).
		Only(ctx)
	if err != nil {
		return err
	}
	if wr.CheckpointData == nil {
		return nil
	}
	raw, err := sonic.Marshal(wr.CheckpointData)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, target)
}

// GetRun returns a workflow run by ID.
func (s *WorkflowRunStore) GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	wr, err := s.client.WorkflowRun.Query().
		Where(workflowrun.IDEQ(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowRunToModel(wr), nil
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running workflow.
func (s *WorkflowRunStore) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetLastHeartbeat(time.Now()).
		Exec(ctx)
}

// genWorkflowRunToModel converts an Ent WorkflowRun entity to the model type.
func genWorkflowRunToModel(wr *gen.WorkflowRun) *model.WorkflowRun {
	return &model.WorkflowRun{
		ID:             wr.ID,
		WorkflowName:   wr.WorkflowName,
		WorkflowFile:   wr.WorkflowFile,
		Status:         model.WorkflowRunState(wr.Status),
		TriggerType:    wr.TriggerType,
		TriggerInfo:    model.JSON(wr.TriggerInfo),
		InputParams:    model.JSON(wr.InputParams),
		CheckpointData: model.JSON(wr.CheckpointData),
		LastHeartbeat:  wr.LastHeartbeat,
		Error:          wr.Error,
		StartedAt:      wr.StartedAt,
		CompletedAt:    wr.CompletedAt,
		CreatedAt:      wr.CreatedAt,
	}
}

// genWorkflowStepRunToModel converts an Ent WorkflowStepRun entity to the model type.
func genWorkflowStepRunToModel(sr *gen.WorkflowStepRun) *model.WorkflowStepRun {
	return &model.WorkflowStepRun{
		ID:            sr.ID,
		WorkflowRunID: sr.WorkflowRunID,
		StepID:        sr.StepID,
		StepName:      sr.StepName,
		Action:        sr.Action,
		ActionType:    sr.ActionType,
		Params:        model.JSON(sr.Params),
		Result:        model.JSON(sr.Result),
		Attempt:       sr.Attempt,
		Status:        model.WorkflowRunState(sr.Status),
		Error:         sr.Error,
		StartedAt:     sr.StartedAt,
		CompletedAt:   sr.CompletedAt,
		CreatedAt:     sr.CreatedAt,
	}
}
