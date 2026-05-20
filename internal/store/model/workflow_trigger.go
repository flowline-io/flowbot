package model

import (
	"time"
)

// WorkflowTrigger mapped from table <workflow_trigger>
type WorkflowTrigger struct {
	ID         int64                `json:"id"`
	WorkflowID int64                `json:"workflow_id"`
	Type       TriggerType          `json:"type"`
	Rule       JSON                 `json:"rule"`
	Count_     int32                `json:"count"`
	State      WorkflowTriggerState `json:"state"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}
