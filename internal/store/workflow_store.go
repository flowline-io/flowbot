package store

import (
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"gorm.io/gorm"
)

// WorkflowStore persists workflow job and step execution state to MySQL.
type WorkflowStore struct {
	db *gorm.DB
}

// NewWorkflowStore creates a WorkflowStore backed by a GORM DB.
func NewWorkflowStore(db *gorm.DB) *WorkflowStore {
	return &WorkflowStore{db: db}
}

// CreateJob inserts a new workflow job in Running state.
func (s *WorkflowStore) CreateJob(uid, topic string, workflowID int64) (*model.Job, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	job := model.Job{
		UID:        uid,
		Topic:      topic,
		WorkflowID: workflowID,
		State:      model.JobRunning,
		StartedAt:  &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.db.Create(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJobState updates the state of a workflow job.
func (s *WorkflowStore) UpdateJobState(jobID int64, state model.JobState, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"state":      state,
		"updated_at": now,
	}
	if state == model.JobSucceeded || state == model.JobCanceled || state == model.JobFailed {
		updates["ended_at"] = now
	}
	return s.db.Model(&model.Job{}).Where("id = ?", jobID).Updates(updates).Error
}

// CreateStep inserts a new workflow step record in Running state.
func (s *WorkflowStore) CreateStep(jobID int64, stepID, name, description string, params types.KV) (*model.Step, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	actionJSON := model.JSON{}
	if params != nil {
		_ = actionJSON.Scan(params)
	}
	step := model.Step{
		JobID:     jobID,
		Action:    actionJSON,
		Name:      name,
		Describe:  description,
		NodeID:    stepID,
		State:     model.StepRunning,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Create(&step).Error; err != nil {
		return nil, err
	}
	return &step, nil
}

// UpdateStepState updates the state, output, and error of a workflow step.
func (s *WorkflowStore) UpdateStepState(stepID int64, state model.StepState, output types.KV, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"state":      state,
		"updated_at": now,
	}
	if state == model.StepSucceeded || state == model.StepFailed || state == model.StepCanceled {
		updates["ended_at"] = now
	}
	if output != nil {
		outJSON := model.JSON{}
		_ = outJSON.Scan(output)
		updates["output"] = outJSON
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return s.db.Model(&model.Step{}).Where("id = ?", stepID).Updates(updates).Error
}

// GetIncompleteJobs returns jobs that are still in a running state.
func (s *WorkflowStore) GetIncompleteJobs() ([]*model.Job, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var jobs []*model.Job
	err := s.db.Where("state = ?", model.JobRunning).
		Order("created_at ASC").
		Find(&jobs).Error
	return jobs, err
}

// GetJobSteps returns all steps for a given job, ordered by creation time.
func (s *WorkflowStore) GetJobSteps(jobID int64) ([]*model.Step, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var steps []*model.Step
	err := s.db.Where("job_id = ?", jobID).
		Order("created_at ASC").
		Find(&steps).Error
	return steps, err
}

// GetStepByNodeID returns the step record for a specific node in a job.
func (s *WorkflowStore) GetStepByNodeID(jobID int64, nodeID string) (*model.Step, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var step model.Step
	err := s.db.Where("job_id = ? AND node_id = ?", jobID, nodeID).First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}
