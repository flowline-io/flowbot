package chatagent

import "errors"

var (
	errCronRequired        = errors.New("cron is required for recurring tasks")
	errRunAtRequired       = errors.New("run_at is required for one-shot tasks")
	errCronRunAtConflict   = errors.New("provide either cron or run_at, not both")
	errInvalidScheduleKind = errors.New("invalid schedule kind")
	errInvalidRunAt        = errors.New("run_at must be ISO8601 UTC (RFC3339)")
	errRunAtPast           = errors.New("run_at must be in the future")
	errTaskIDRequired      = errors.New("task_id is required")
	errNoUpdateFields      = errors.New("at least one updatable field is required")
	errInvalidTaskState    = errors.New("state must be active or paused")
	errKindImmutable       = errors.New("schedule kind cannot be changed; cancel and recreate the task")
	errWrongKindCron       = errors.New("cron can only be updated on recurring tasks")
	errWrongKindOnce       = errors.New("run_at can only be updated on one-shot tasks")
)
