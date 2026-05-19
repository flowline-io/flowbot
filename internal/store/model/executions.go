package model

import (
	"time"
)


// Execution mapped from table <executions>
type Execution struct {
	ID          int64          `json:"id"`
	FlowID      int64          `json:"flow_id"`
	ExecutionID string         `json:"execution_id"`
	TriggerType string         `json:"trigger_type"`
	TriggerID   string         `json:"trigger_id"`
	State       ExecutionState `json:"state"`
	Payload     JSON           `json:"payload"`
	Variables   JSON           `json:"variables"`
	Result      JSON           `json:"result"`
	Error       string         `json:"error"`
	StartedAt   *time.Time     `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
