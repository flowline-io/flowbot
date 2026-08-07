package types

import "time"

// GatewayCLI identifies which local CLI a gateway job should run.
type GatewayCLI string

const (
	GatewayCLICursor   GatewayCLI = "cursor"
	GatewayCLIOpenCode GatewayCLI = "opencode"
)

// GatewayJobStatus is the lifecycle state of a local-CLI gateway job.
type GatewayJobStatus string

const (
	GatewayJobPending   GatewayJobStatus = "pending"
	GatewayJobRunning   GatewayJobStatus = "running"
	GatewayJobSucceeded GatewayJobStatus = "succeeded"
	GatewayJobFailed    GatewayJobStatus = "failed"
	GatewayJobCanceled  GatewayJobStatus = "canceled"
)

// GatewayJobTerminal reports whether status is a terminal job state.
func GatewayJobTerminal(status GatewayJobStatus) bool {
	switch status {
	case GatewayJobSucceeded, GatewayJobFailed, GatewayJobCanceled:
		return true
	default:
		return false
	}
}

// GatewayCreateJob is the input for creating a gateway job on the server.
type GatewayCreateJob struct {
	UID    string     `json:"uid"`
	CLI    GatewayCLI `json:"cli"`
	Prompt string     `json:"prompt"`
	Cwd    string     `json:"cwd,omitempty"`
}

// GatewayClaimRequest is the body for POST /gateway/v1/claim.
type GatewayClaimRequest struct {
	WorkerID string `json:"worker_id"`
}

// GatewayClaimResponse is returned when a job is claimed (or empty when none).
type GatewayClaimResponse struct {
	Job *GatewayJob `json:"job,omitempty"`
}

// GatewayCompleteRequest is the body for POST /gateway/v1/jobs/{id}/result.
type GatewayCompleteRequest struct {
	WorkerID   string           `json:"worker_id"`
	Status     GatewayJobStatus `json:"status"`
	Output     string           `json:"output,omitempty"`
	ExitCode   *int             `json:"exit_code,omitempty"`
	Error      string           `json:"error,omitempty"`
	DurationMs int64            `json:"duration_ms,omitempty"`
	Cwd        string           `json:"cwd,omitempty"`
}

// GatewayHeartbeatRequest is the body for POST /gateway/v1/heartbeat.
type GatewayHeartbeatRequest struct {
	WorkerID string `json:"worker_id"`
	JobID    string `json:"job_id,omitempty"`
}

// GatewayJob is the shared job view for store, HTTP, and capability.
type GatewayJob struct {
	JobID      string           `json:"job_id"`
	UID        string           `json:"uid,omitempty"`
	CLI        GatewayCLI       `json:"cli"`
	Prompt     string           `json:"prompt"`
	Cwd        string           `json:"cwd,omitempty"`
	Status     GatewayJobStatus `json:"status"`
	Output     string           `json:"output,omitempty"`
	ExitCode   *int             `json:"exit_code,omitempty"`
	Error      string           `json:"error,omitempty"`
	Truncated  bool             `json:"truncated,omitempty"`
	DurationMs int64            `json:"duration_ms,omitempty"`
	WorkerID   string           `json:"worker_id,omitempty"`
	LeaseUntil *time.Time       `json:"lease_until,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	ClaimedAt  *time.Time       `json:"claimed_at,omitempty"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
}
