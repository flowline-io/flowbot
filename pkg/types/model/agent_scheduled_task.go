package model

import "time"

// AgentScheduledTask represents one scheduled task for UI display and transport.
type AgentScheduledTask struct {
	TaskID          string     `json:"task_id"`
	Name            string     `json:"name"`
	ScheduleKind    string     `json:"schedule_kind"`
	Cron            string     `json:"cron"`
	RunAt           *time.Time `json:"run_at"`
	Prompt          string     `json:"prompt"`
	State           string     `json:"state"`
	SourceSessionID string     `json:"source_session_id"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentScheduledTaskRun represents one scheduled task execution for UI display.
type AgentScheduledTaskRun struct {
	RunID        string     `json:"run_id"`
	TaskID       string     `json:"task_id"`
	RunSessionID string     `json:"run_session_id"`
	State        string     `json:"state"`
	Reply        string     `json:"reply"`
	Error        string     `json:"error"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}
