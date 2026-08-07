package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Cursor runs the Cursor headless CLI (agent -p ...).
type Cursor struct {
	Binary  string
	APIKey  string
	Timeout time.Duration
}

// NewCursor builds a Cursor runner.
func NewCursor(binary, apiKey string, timeout time.Duration) *Cursor {
	if binary == "" {
		binary = "agent"
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return &Cursor{Binary: binary, APIKey: apiKey, Timeout: timeout}
}

// Run implements Runner.
func (c *Cursor) Run(ctx context.Context, job *types.GatewayJob, workspace string) Result {
	if job == nil {
		return Result{ExitCode: 1, Err: errors.New("nil job")}
	}
	runCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	args := []string{
		"-p", "--force", "--trust",
		"--workspace", workspace,
		"--output-format", "text",
		job.Prompt,
	}
	cmd := exec.CommandContext(runCtx, c.Binary, args...)
	cmd.Dir = workspace
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if c.APIKey != "" {
		cmd.Env = append(os.Environ(), "CURSOR_API_KEY="+c.APIKey)
	} else {
		cmd.Env = os.Environ()
	}
	flog.Info("starting cursor cli job_id=%s binary=%s workspace=%s timeout=%s",
		job.JobID, c.Binary, workspace, c.Timeout)
	start := time.Now()
	err := cmd.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		if out != "" {
			out += "\n"
		}
		out += stderr.String()
	}
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			code = 1
			err = fmt.Errorf("cursor cli: %w", err)
		}
		flog.Warn("cursor cli finished with error job_id=%s exit_code=%d duration_ms=%d stdout_len=%d stderr_len=%d err=%v output=%q",
			job.JobID, code, time.Since(start).Milliseconds(), stdout.Len(), stderr.Len(), err, truncateForLog(out, 512))
	} else {
		flog.Info("cursor cli finished job_id=%s exit_code=%d duration_ms=%d stdout_len=%d stderr_len=%d",
			job.JobID, code, time.Since(start).Milliseconds(), stdout.Len(), stderr.Len())
	}
	return Result{Output: out, ExitCode: code, Err: err}
}

func truncateForLog(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
