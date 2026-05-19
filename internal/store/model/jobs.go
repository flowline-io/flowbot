package model

import (
	"time"
)

// Job mapped from table <jobs>
type Job struct {
	ID            int64      `json:"id"`
	UID           string     `json:"uid"`
	Topic         string     `json:"topic"`
	WorkflowID    int64      `json:"workflow_id"`
	DagID         int64      `json:"dag_id"`
	TriggerID     int64      `json:"trigger_id"`
	ScriptVersion int32      `json:"script_version"`
	State         JobState   `json:"state"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Steps         []*Step    `json:"steps"`
}
