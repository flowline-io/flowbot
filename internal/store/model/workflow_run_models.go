package model

import "time"

const TableNameWorkflowRun = "workflow_runs"

// WorkflowRun records a single execution of a local workflow engine workflow.
type WorkflowRun struct {
	ID             int64            `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	WorkflowName   string           `gorm:"column:workflow_name;not null;index" json:"workflow_name"`
	WorkflowFile   string           `gorm:"column:workflow_file;not null" json:"workflow_file"`
	Status         WorkflowRunState `gorm:"column:status;not null;default:0" json:"status"`
	TriggerType    string           `gorm:"column:trigger_type;default:''" json:"trigger_type"`
	TriggerInfo    JSON             `gorm:"column:trigger_info" json:"trigger_info,omitempty"`
	InputParams    JSON             `gorm:"column:input_params" json:"input_params,omitempty"`
	CheckpointData JSON             `gorm:"column:checkpoint_data" json:"checkpoint_data,omitempty"`
	LastHeartbeat  *time.Time       `gorm:"column:last_heartbeat" json:"last_heartbeat,omitempty"`
	Error          string           `gorm:"column:error" json:"error,omitempty"`
	StartedAt      time.Time        `gorm:"column:started_at" json:"started_at"`
	CompletedAt    *time.Time       `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt      time.Time        `gorm:"column:created_at;not null" json:"created_at"`
}

func (*WorkflowRun) TableName() string {
	return TableNameWorkflowRun
}

const TableNameWorkflowStepRun = "workflow_step_runs"

// WorkflowStepRun records a single step execution within a workflow run.
type WorkflowStepRun struct {
	ID             int64            `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	WorkflowRunID  int64            `gorm:"column:workflow_run_id;not null;index" json:"workflow_run_id"`
	StepID         string           `gorm:"column:step_id;not null" json:"step_id"`
	StepName       string           `gorm:"column:step_name;default:''" json:"step_name"`
	Action         string           `gorm:"column:action;not null" json:"action"`
	ActionType     string           `gorm:"column:action_type;not null" json:"action_type"`
	Params         JSON             `gorm:"column:params" json:"params,omitempty"`
	Result         JSON             `gorm:"column:result" json:"result,omitempty"`
	Attempt        int              `gorm:"column:attempt;not null;default:1" json:"attempt"`
	Status         WorkflowRunState `gorm:"column:status;not null;default:0" json:"status"`
	Error          string           `gorm:"column:error" json:"error,omitempty"`
	StartedAt      time.Time        `gorm:"column:started_at" json:"started_at"`
	CompletedAt    *time.Time       `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt      time.Time        `gorm:"column:created_at;not null" json:"created_at"`
}

func (*WorkflowStepRun) TableName() string {
	return TableNameWorkflowStepRun
}
