package model

import (
	"time"
)

// Workflow mapped from table <workflow>
type Workflow struct {
	ID              int64              `json:"id"`
	UID             string             `json:"uid"`
	Topic           string             `json:"topic"`
	Flag            string             `json:"flag"`
	Name            string             `json:"name"`
	Describe        string             `json:"describe"`
	SuccessfulCount int32              `json:"successful_count"`
	FailedCount     int32              `json:"failed_count"`
	RunningCount    int32              `json:"running_count"`
	CanceledCount   int32              `json:"canceled_count"`
	State           WorkflowState      `json:"state"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Dag             []*Dag             `json:"dag"`
	Triggers        []*WorkflowTrigger `json:"triggers"`
}
