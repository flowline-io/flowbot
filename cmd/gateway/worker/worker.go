// Package worker claims and executes CapGateway jobs on the local machine.
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/cmd/gateway/client"
	"github.com/flowline-io/flowbot/cmd/gateway/config"
	"github.com/flowline-io/flowbot/cmd/gateway/cwd"
	"github.com/flowline-io/flowbot/cmd/gateway/runner"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Worker claims and executes gateway jobs.
type Worker struct {
	api     *client.Client
	cfg     *config.Config
	runners map[types.GatewayCLI]runner.Runner
	sem     chan struct{}
}

// New creates a Worker.
func New(api *client.Client, cfg *config.Config, runners map[types.GatewayCLI]runner.Runner) *Worker {
	n := cfg.MaxConcurrent
	if n <= 0 {
		n = 1
	}
	return &Worker{
		api:     api,
		cfg:     cfg,
		runners: runners,
		sem:     make(chan struct{}, n),
	}
}

// Run loops until ctx is canceled.
func (w *Worker) Run(ctx context.Context) error {
	claimTicker := time.NewTicker(w.cfg.ClaimInterval)
	defer claimTicker.Stop()
	hbTicker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer hbTicker.Stop()

	if err := w.api.Heartbeat(ctx, w.cfg.WorkerID, ""); err != nil {
		flog.Warn("initial heartbeat failed: %v", err)
	} else {
		flog.Info("initial heartbeat ok worker_id=%s", w.cfg.WorkerID)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			flog.Info("shutdown requested, waiting for in-flight jobs")
			return ctx.Err()
		case <-hbTicker.C:
			if err := w.api.Heartbeat(ctx, w.cfg.WorkerID, ""); err != nil {
				flog.Warn("heartbeat failed: %v", err)
			} else {
				flog.Debug("heartbeat ok worker_id=%s", w.cfg.WorkerID)
			}
		case <-claimTicker.C:
			select {
			case w.sem <- struct{}{}:
			default:
				flog.Debug("skip claim: at max_concurrent")
				continue
			}
			job, err := w.api.Claim(ctx, w.cfg.WorkerID)
			if err != nil {
				<-w.sem
				flog.Warn("claim failed: %v", err)
				continue
			}
			if job == nil {
				<-w.sem
				flog.Debug("claim empty")
				continue
			}
			flog.Info("claimed job job_id=%s cli=%s cwd=%s prompt_len=%d",
				job.JobID, job.CLI, job.Cwd, len(job.Prompt))
			wg.Add(1)
			go func(j *types.GatewayJob) {
				defer wg.Done()
				defer func() { <-w.sem }()
				w.execute(ctx, j)
			}(job)
		}
	}
}

func (w *Worker) execute(parent context.Context, job *types.GatewayJob) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan struct{})
	go w.watchCancel(ctx, cancel, done, job.JobID)
	defer close(done)

	start := time.Now()
	workspace, err := cwd.Resolve(job.Cwd, w.cfg.DefaultWorkspace, w.cfg.WorkspaceAllowlist)
	if err != nil {
		flog.Error(fmt.Errorf("cwd resolve failed job_id=%s cwd=%s: %w", job.JobID, job.Cwd, err))
		w.fail(parent, job.JobID, "", err.Error(), start)
		return
	}
	r := w.runners[job.CLI]
	if r == nil {
		msg := fmt.Sprintf("no runner for cli %q", job.CLI)
		flog.Error(fmt.Errorf("runner missing job_id=%s cli=%s", job.JobID, job.CLI))
		w.fail(parent, job.JobID, workspace, msg, start)
		return
	}
	flog.Info("running job job_id=%s cli=%s workspace=%s", job.JobID, job.CLI, workspace)
	res := r.Run(ctx, job, workspace)
	w.reportResult(parent, ctx, job, workspace, res, start)
}

func (w *Worker) watchCancel(ctx context.Context, cancel context.CancelFunc, done <-chan struct{}, jobID string) {
	hbTicker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer hbTicker.Stop()
	// Poll cancel more often than lease heartbeat so CapGateway cancel stops the CLI quickly.
	cancelEvery := 2 * time.Second
	if w.cfg.ClaimInterval > 0 && w.cfg.ClaimInterval < cancelEvery {
		cancelEvery = w.cfg.ClaimInterval
	}
	cancelTicker := time.NewTicker(cancelEvery)
	defer cancelTicker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-hbTicker.C:
			if err := w.api.Heartbeat(ctx, w.cfg.WorkerID, jobID); err != nil {
				flog.Warn("job heartbeat failed job_id=%s: %v", jobID, err)
			}
		case <-cancelTicker.C:
			cur, err := w.api.GetJob(ctx, jobID)
			if err != nil {
				flog.Debug("get job during watch failed job_id=%s: %v", jobID, err)
				continue
			}
			if cur != nil && cur.Status == types.GatewayJobCanceled {
				flog.Info("job canceled by server, stopping CLI job_id=%s", jobID)
				cancel()
				return
			}
		}
	}
}

func (w *Worker) fail(ctx context.Context, jobID, workspace, errText string, start time.Time) {
	dur := time.Since(start).Milliseconds()
	if err := w.api.Complete(ctx, jobID, types.GatewayCompleteRequest{
		WorkerID: w.cfg.WorkerID, Status: types.GatewayJobFailed, Error: errText, Cwd: workspace,
		DurationMs: dur,
	}); err != nil {
		flog.Error(fmt.Errorf("complete failed job report failed job_id=%s: %w", jobID, err))
		return
	}
	flog.Error(fmt.Errorf("job failed job_id=%s error=%s duration_ms=%d cwd=%s", jobID, errText, dur, workspace))
}

func (w *Worker) reportResult(parent, runCtx context.Context, job *types.GatewayJob, workspace string, res runner.Result, start time.Time) {
	status := types.GatewayJobSucceeded
	errText := ""
	if res.Err != nil || res.ExitCode != 0 {
		status = types.GatewayJobFailed
		if res.Err != nil {
			errText = res.Err.Error()
		}
	}
	if runCtx.Err() != nil {
		cur, gerr := w.api.GetJob(parent, job.JobID)
		if gerr == nil && cur != nil && cur.Status == types.GatewayJobCanceled {
			flog.Info("job ended as canceled job_id=%s duration_ms=%d", job.JobID, time.Since(start).Milliseconds())
			return
		}
		status = types.GatewayJobFailed
		if errText == "" {
			errText = "canceled or interrupted"
		}
	}
	code := res.ExitCode
	dur := time.Since(start).Milliseconds()
	if err := w.api.Complete(parent, job.JobID, types.GatewayCompleteRequest{
		WorkerID: w.cfg.WorkerID, Status: status, Output: res.Output, ExitCode: &code,
		Error: errText, DurationMs: dur, Cwd: workspace,
	}); err != nil {
		flog.Error(fmt.Errorf("complete result report failed job_id=%s: %w", job.JobID, err))
		return
	}
	if status == types.GatewayJobSucceeded {
		flog.Info("job succeeded job_id=%s exit_code=%d duration_ms=%d output_len=%d",
			job.JobID, code, dur, len(res.Output))
		return
	}
	flog.Error(fmt.Errorf("job failed job_id=%s exit_code=%d error=%s duration_ms=%d output_len=%d output=%q",
		job.JobID, code, errText, dur, len(res.Output), truncateJobLog(res.Output, 512)))
}

func truncateJobLog(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
