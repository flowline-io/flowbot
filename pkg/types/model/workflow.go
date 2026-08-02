package model

import "time"

// Workflow is a workflow definition row for UI and catalog use.
type Workflow struct {
	ID             int64            `json:"id"`
	Name           string           `json:"name"`
	Describe       string           `json:"describe,omitempty"`
	Enabled        bool             `json:"enabled"`
	Resumable      bool             `json:"resumable,omitempty"`
	MaxConcurrency int              `json:"max_concurrency,omitempty"`
	Pipeline       []string         `json:"pipeline,omitempty"`
	Inputs         []map[string]any `json:"inputs,omitempty"`
}

// WorkflowTrigger is a workflow trigger row for UI and metadata rebuild.
type WorkflowTrigger struct {
	ID         int64          `json:"id"`
	WorkflowID int64          `json:"workflow_id"`
	Type       string         `json:"type"`
	Enabled    bool           `json:"enabled"`
	Rule       map[string]any `json:"rule,omitempty"`
}

// WorkflowTaskRow is a normalized workflow task row used to rebuild metadata.
type WorkflowTaskRow struct {
	TaskID   string         `json:"task_id"`
	Action   string         `json:"action"`
	Describe string         `json:"describe,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	Vars     []string       `json:"vars,omitempty"`
	Conn     []string       `json:"conn,omitempty"`
	Retry    map[string]any `json:"retry,omitempty"`
}

// WorkflowRun is a workflow run row for UI and engine persistence.
type WorkflowRun struct {
	ID           int64      `json:"id"`
	WorkflowID   *int64     `json:"workflow_id,omitempty"`
	WorkflowName string     `json:"workflow_name,omitempty"`
	Status       int        `json:"status"`
	TriggerType  string     `json:"trigger_type,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// WorkflowStepRun is a workflow step run row for UI and engine persistence.
type WorkflowStepRun struct {
	ID          int64          `json:"id,omitempty"`
	StepID      string         `json:"step_id"`
	StepName    string         `json:"step_name"`
	Action      string         `json:"action,omitempty"`
	ActionType  string         `json:"action_type,omitempty"`
	Attempt     int            `json:"attempt,omitempty"`
	Status      int            `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
}
