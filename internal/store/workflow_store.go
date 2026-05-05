package store

import (
	"encoding/json"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"gorm.io/gorm"
)

// WorkflowRunStore persists workflow runs, step runs, and checkpoint data.
type WorkflowRunStore struct {
	db *gorm.DB
}

// NewWorkflowRunStore creates a WorkflowRunStore backed by the given database.
func NewWorkflowRunStore(db *gorm.DB) *WorkflowRunStore {
	return &WorkflowRunStore{db: db}
}

// CreateRun inserts a new workflow run record.
func (s *WorkflowRunStore) CreateRun(workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	run := model.WorkflowRun{
		WorkflowName: workflowName,
		WorkflowFile: workflowFile,
		Status:       model.WorkflowRunRunning,
		TriggerType:  triggerType,
		TriggerInfo:  triggerInfo,
		InputParams:  inputParams,
		StartedAt:    now,
		CreatedAt:    now,
	}
	if err := s.db.Create(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// UpdateRunStatus updates the status, error, and completed_at of a workflow run.
func (s *WorkflowRunStore) UpdateRunStatus(runID int64, status model.WorkflowRunState, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"completed_at": now,
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return s.db.Model(&model.WorkflowRun{}).Where("id = ?", runID).Updates(updates).Error
}

// CreateStepRun inserts a new workflow step run record.
func (s *WorkflowRunStore) CreateStepRun(runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	sr := model.WorkflowStepRun{
		WorkflowRunID: runID,
		StepID:        stepID,
		StepName:      stepName,
		Action:        action,
		ActionType:    actionType,
		Params:        params,
		Attempt:       attempt,
		Status:        model.WorkflowRunRunning,
		StartedAt:     now,
		CreatedAt:     now,
	}
	if err := s.db.Create(&sr).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

// UpdateStepRun updates the status, result, error, and attempt count of a workflow step run.
// completed_at is only set for terminal states (Done, Failed).
func (s *WorkflowRunStore) UpdateStepRun(stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.db == nil {
		return nil
	}
	updates := map[string]any{
		"status":  status,
		"attempt": attempt,
	}
	if status == model.WorkflowRunDone || status == model.WorkflowRunFailed {
		now := time.Now()
		updates["completed_at"] = now
	}
	if result != nil {
		updates["result"] = result
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return s.db.Model(&model.WorkflowStepRun{}).Where("id = ?", stepRunID).Updates(updates).Error
}

// SaveCheckpoint persists the intermediate workflow run state.
func (s *WorkflowRunStore) SaveCheckpoint(runID int64, data any) error {
	if s == nil || s.db == nil {
		return nil
	}
	cp := model.JSON{}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := cp.Scan(raw); err != nil {
		return err
	}
	return s.db.Model(&model.WorkflowRun{}).
		Where("id = ?", runID).
		Update("checkpoint_data", cp).Error
}

// GetIncompleteRuns returns workflow runs that are still running and may need recovery.
func (s *WorkflowRunStore) GetIncompleteRuns() ([]*model.WorkflowRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var runs []*model.WorkflowRun
	err := s.db.Where("status = ?", model.WorkflowRunRunning).
		Order("created_at ASC").
		Find(&runs).Error
	return runs, err
}

// GetCheckpoint loads the checkpoint data for a workflow run.
func (s *WorkflowRunStore) GetCheckpoint(runID int64, target any) error {
	if s == nil || s.db == nil {
		return nil
	}
	var run model.WorkflowRun
	if err := s.db.Select("checkpoint_data").Where("id = ?", runID).First(&run).Error; err != nil {
		return err
	}
	if run.CheckpointData == nil {
		return nil
	}
	raw, err := json.Marshal(run.CheckpointData)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// GetRun returns a workflow run by ID.
func (s *WorkflowRunStore) GetRun(runID int64) (*model.WorkflowRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var run model.WorkflowRun
	if err := s.db.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running workflow.
func (s *WorkflowRunStore) UpdateRunHeartbeat(runID int64) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	return s.db.Model(&model.WorkflowRun{}).
		Where("id = ?", runID).
		Update("last_heartbeat", now).Error
}
