package workflow

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
)

// CheckpointData is the intermediate state saved at each workflow step boundary.
type CheckpointData struct {
	StepIndex      int               `json:"step_index"`
	CompletedTasks map[string]bool   `json:"completed_tasks"`
	StepResults    map[string]string `json:"step_results"`
	Input          types.KV          `json:"input"`
	HeartbeatAt    time.Time         `json:"heartbeat_at"`
}

// WorkflowRunStore persists workflow runs, step runs, and checkpoint data.
type WorkflowRunStore interface {
	CreateRun(ctx context.Context, workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error)
	UpdateRunStatus(ctx context.Context, runID int64, status model.WorkflowRunState, errMsg string) error
	CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error)
	UpdateStepRun(ctx context.Context, stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(ctx context.Context, runID int64, data any) error
	GetIncompleteRuns(ctx context.Context) ([]*model.WorkflowRun, error)
	GetCheckpoint(ctx context.Context, runID int64, target any) error
	GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error)
	UpdateRunHeartbeat(ctx context.Context, runID int64) error
}
