package model

import "time"

// WorkflowRun records a single execution of a local workflow engine workflow.
type WorkflowRun struct {
	ID             int64            `json:"id"`
	WorkflowName   string           `json:"workflow_name"`
	WorkflowFile   string           `json:"workflow_file"`
	Status         WorkflowRunState `json:"status"`
	TriggerType    string           `json:"trigger_type"`
	TriggerInfo    JSON             `json:"trigger_info,omitempty"`
	InputParams    JSON             `json:"input_params,omitempty"`
	CheckpointData JSON             `json:"checkpoint_data,omitempty"`
	LastHeartbeat  *time.Time       `json:"last_heartbeat,omitempty"`
	Error          string           `json:"error,omitempty"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

// WorkflowStepRun records a single step execution within a workflow run.
type WorkflowStepRun struct {
	ID            int64            `json:"id"`
	WorkflowRunID int64            `json:"workflow_run_id"`
	StepID        string           `json:"step_id"`
	StepName      string           `json:"step_name"`
	Action        string           `json:"action"`
	ActionType    string           `json:"action_type"`
	Params        JSON             `json:"params,omitempty"`
	Result        JSON             `json:"result,omitempty"`
	Attempt       int              `json:"attempt"`
	Status        WorkflowRunState `json:"status"`
	Error         string           `json:"error,omitempty"`
	StartedAt     time.Time        `json:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}
