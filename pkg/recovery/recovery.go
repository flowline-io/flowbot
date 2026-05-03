// Package recovery handles restart recovery for pipelines and workflows.
// It scans for incomplete runs at startup and resumes them when possible.
package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// RunStore is the subset of pipeline.RunStore needed for recovery.
type RunStore interface {
	GetIncompleteRuns() (runs []*model.PipelineRun, err error)
	GetCheckpoint(runID int64, target any) error
	UpdateRunStatus(runID int64, status model.PipelineState, errMsg string) error
	UpdateRunHeartbeat(runID int64) error
}

// WorkflowRecoveryStore is the subset needed for workflow recovery.
type WorkflowRecoveryStore interface {
	GetIncompleteJobs() (jobs []*model.Job, err error)
	GetJobSteps(jobID int64) (steps []*model.Step, err error)
	UpdateJobState(jobID int64, state model.JobState, errMsg string) error
}

// Config controls recovery behavior.
type Config struct {
	Enabled      bool          `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	StaleTimeout time.Duration `json:"stale_timeout" yaml:"stale_timeout" mapstructure:"stale_timeout"`
	AutoResume   bool          `json:"auto_resume" yaml:"auto_resume" mapstructure:"auto_resume"`
	MaxResumeAge time.Duration `json:"max_resume_age" yaml:"max_resume_age" mapstructure:"max_resume_age"`
}

// Manager orchestrates restart recovery for pipelines and workflows.
type Manager struct {
	cfg           Config
	pipelineStore RunStore
	workflowStore WorkflowRecoveryStore
}

// New creates a new recovery Manager.
func New(cfg Config, pipelineStore RunStore, workflowStore WorkflowRecoveryStore) *Manager {
	return &Manager{
		cfg:           cfg,
		pipelineStore: pipelineStore,
		workflowStore: workflowStore,
	}
}

// Recover scans for incomplete runs and attempts to resume them if configured.
func (m *Manager) Recover(ctx context.Context) error {
	if !m.cfg.Enabled {
		flog.Info("recovery manager disabled")
		return nil
	}

	flog.Info("recovery manager starting scan")
	pipelineRecovered, pipelineErr := m.recoverPipelines(ctx)
	workflowRecovered, workflowErr := m.recoverWorkflows(ctx)

	flog.Info("recovery scan complete: pipelines=%d workflows=%d",
		pipelineRecovered, workflowRecovered)

	if pipelineErr != nil {
		return fmt.Errorf("pipeline recovery: %w", pipelineErr)
	}
	if workflowErr != nil {
		return fmt.Errorf("workflow recovery: %w", workflowErr)
	}
	return nil
}

func (m *Manager) recoverPipelines(ctx context.Context) (int, error) {
	if m.pipelineStore == nil {
		return 0, nil
	}

	runs, err := m.pipelineStore.GetIncompleteRuns()
	if err != nil {
		return 0, fmt.Errorf("get incomplete pipeline runs: %w", err)
	}

	if len(runs) == 0 {
		return 0, nil
	}

	flog.Info("found %d incomplete pipeline runs", len(runs))
	recovered := 0

	for _, run := range runs {
		if !m.isStale(run) {
			flog.Info("pipeline run %d (%s) is still active", run.ID, run.PipelineName)
			continue
		}

		if m.cfg.MaxResumeAge > 0 && time.Since(run.StartedAt) > m.cfg.MaxResumeAge {
			flog.Info("pipeline run %d (%s) exceeded max resume age, marking as cancelled",
				run.ID, run.PipelineName)
			_ = m.pipelineStore.UpdateRunStatus(run.ID, model.PipelineCancel, "exceeded max resume age")
			continue
		}

		if !m.cfg.AutoResume {
			flog.Info("pipeline run %d (%s) is stale but auto-resume disabled, marking as cancelled",
				run.ID, run.PipelineName)
			_ = m.pipelineStore.UpdateRunStatus(run.ID, model.PipelineCancel, "stale run, auto-resume disabled")
			continue
		}

		flog.Info("pipeline run %d (%s) marked for resume", run.ID, run.PipelineName)
		// The actual resume logic must be wired externally since pipeline execution
		// depends on the engine, definitions, and ability registry.
		// The recovery manager only identifies stale runs for external resumption.
		recovered++
	}

	return recovered, nil
}

func (m *Manager) recoverWorkflows(ctx context.Context) (int, error) {
	if m.workflowStore == nil {
		return 0, nil
	}

	jobs, err := m.workflowStore.GetIncompleteJobs()
	if err != nil {
		return 0, fmt.Errorf("get incomplete workflow jobs: %w", err)
	}

	if len(jobs) == 0 {
		return 0, nil
	}

	flog.Info("found %d incomplete workflow jobs", len(jobs))
	recovered := 0

	for _, job := range jobs {
		if !m.cfg.AutoResume {
			flog.Info("workflow job %d auto-resume disabled, marking as failed", job.ID)
			_ = m.workflowStore.UpdateJobState(job.ID, model.JobFailed, "stale job, auto-resume disabled")
			continue
		}

		flog.Info("workflow job %d marked for resume", job.ID)
		recovered++
	}

	return recovered, nil
}

func (m *Manager) isStale(run *model.PipelineRun) bool {
	if m.cfg.StaleTimeout <= 0 {
		return true
	}
	if run.LastHeartbeat == nil {
		return time.Since(run.StartedAt) > m.cfg.StaleTimeout
	}
	return time.Since(*run.LastHeartbeat) > m.cfg.StaleTimeout
}

// GetIncompletePipelines returns a list of incomplete pipeline runs for admin endpoints.
func (m *Manager) GetIncompletePipelines() ([]*model.PipelineRun, error) {
	if m.pipelineStore == nil {
		return nil, nil
	}
	return m.pipelineStore.GetIncompleteRuns()
}

// GetIncompleteWorkflows returns a list of incomplete workflow jobs for admin endpoints.
func (m *Manager) GetIncompleteWorkflows() ([]*model.Job, error) {
	if m.workflowStore == nil {
		return nil, nil
	}
	return m.workflowStore.GetIncompleteJobs()
}
