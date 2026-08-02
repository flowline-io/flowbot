package types

import "database/sql/driver"

// PipelineState is the execution state of a pipeline run or step.
type PipelineState int

const (
	PipelineStateUnknown PipelineState = iota
	PipelineStart
	PipelineDone
	PipelineCancel
	PipelineFailed
)

// Value implements driver.Valuer for database persistence.
func (j PipelineState) Value() (driver.Value, error) {
	return int64(j), nil
}

// WorkflowRunState is the execution state of a workflow run or step.
type WorkflowRunState int

const (
	WorkflowRunStateUnknown WorkflowRunState = iota
	WorkflowRunRunning
	WorkflowRunDone
	WorkflowRunFailed
)

// Value implements driver.Valuer for database persistence.
func (j WorkflowRunState) Value() (driver.Value, error) {
	return int64(j), nil
}
