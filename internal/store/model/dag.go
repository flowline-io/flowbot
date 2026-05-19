package model

import (
	"time"
)

// Dag mapped from table <dag>
type Dag struct {
	ID            int64     `json:"id"`
	WorkflowID    int64     `json:"workflow_id"`
	ScriptID      int64     `json:"script_id"`
	ScriptVersion int32     `json:"script_version"`
	Nodes         []*Node   `json:"nodes"`
	Edges         []*Edge   `json:"edges"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
