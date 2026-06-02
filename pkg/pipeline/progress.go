// Package pipeline provides the workflow execution engine.
package pipeline

import (
	"context"
	"fmt"
	"time"
)

// StepCallback receives progress events during pipeline execution.
// All methods are called synchronously from the step execution loop.
// nil receiver is safe — Engine skips calls when callback is nil.
type StepCallback interface {
	OnRunStart(ctx context.Context, runID int64, pipelineName string,
		trigger string, totalSteps int, stepNames []string)
	OnStepStart(ctx context.Context, runID int64, pipelineName string,
		stepIndex int, stepName string, input map[string]any)
	OnStepDone(ctx context.Context, runID int64, pipelineName string,
		stepIndex int, stepName string, output map[string]any, elapsedMs int64)
	OnStepError(ctx context.Context, runID int64, pipelineName string,
		stepIndex int, stepName string, err error, elapsedMs int64)
	OnRunComplete(ctx context.Context, runID int64, pipelineName string,
		elapsedMs int64, failed bool, errMsg string)
}

// StepProgressEvent is the JSON payload for a single progress update.
// StepIndex of -1 indicates a run-level event (start/complete/failed).
type StepProgressEvent struct {
	RunID        int64          `json:"run_id"`
	PipelineName string         `json:"pipeline_name"`
	StepIndex    int            `json:"step_index"`
	StepName     string         `json:"step_name"`
	Status       string         `json:"status"`
	Input        map[string]any `json:"input,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	ElapsedMs    int64          `json:"elapsed_ms,omitempty"`
	Error        string         `json:"error,omitempty"`
	TotalSteps   int            `json:"total_steps,omitempty"`
}

// StreamName returns the Redis Stream name for a given run ID.
func StreamName(runID int64) string {
	return fmt.Sprintf("pipeline:run:%d", runID)
}

// StreamTTLFailsafe is the TTL set on stream creation to prevent leaks on crash.
const StreamTTLFailsafe = 24 * time.Hour

// StreamTTLDrain is the TTL after completion for SSE clients to drain.
const StreamTTLDrain = 5 * time.Minute
