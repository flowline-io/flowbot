package workflow

import (
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
)

// CheckpointData is the intermediate state saved at each workflow step boundary.
type CheckpointData struct {
	StepIndex   int               `json:"step_index"`
	StepResults map[string]string `json:"step_results"`
	Input       types.KV          `json:"input"`
	HeartbeatAt time.Time         `json:"heartbeat_at"`
}

// WorkflowRunStore persists workflow runs, step runs, and checkpoint data.
type WorkflowRunStore interface {
	CreateRun(workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error)
	UpdateRunStatus(runID int64, status model.WorkflowRunState, errMsg string) error
	CreateStepRun(runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error)
	UpdateStepRun(stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(runID int64, data any) error
	GetIncompleteRuns() ([]*model.WorkflowRun, error)
	GetCheckpoint(runID int64, target any) error
	GetRun(runID int64) (*model.WorkflowRun, error)
	UpdateRunHeartbeat(runID int64) error
}
