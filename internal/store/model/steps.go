package model

import (
	"time"
)

// Step mapped from table <steps>
type Step struct {
	ID        int64      `json:"id"`
	UID       string     `json:"uid"`
	Topic     string     `json:"topic"`
	JobID     int64      `json:"job_id"`
	Action    JSON       `json:"action"`
	Name      string     `json:"name"`
	Describe  string     `json:"describe"`
	NodeID    string     `json:"node_id"`
	Depend    []string   `json:"depend"`
	Input     JSON       `json:"input"`
	Output    JSON       `json:"output"`
	Error     string     `json:"error"`
	State     StepState  `json:"state"`
	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
