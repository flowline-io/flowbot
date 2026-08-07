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

// Cursor runs a headless coding CLI binary with Cursor-compatible flags.
// It may invoke Cursor's `agent` (CURSOR_API_KEY) or flowbot-agent
// (FLOWBOT_URL + FLOWBOT_AGENT_TOKEN) depending on Binary and child env.
type Cursor struct {
	Binary string
	// APIKey is CURSOR_API_KEY for the upstream Cursor CLI.
	APIKey string
	// FlowbotURL is injected for flowbot-agent as FLOWBOT_URL.
	FlowbotURL string
	// AgentAccessToken is injected for flowbot-agent as FLOWBOT_AGENT_TOKEN (agent:headless).
	AgentAccessToken string
	Timeout          time.Duration
}

// NewCursor builds a Cursor/flowbot-agent runner.
func NewCursor(binary, apiKey string, timeout time.Duration) *Cursor {
	if binary == "" {
		binary = "agent"
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return &Cursor{Binary: binary, APIKey: apiKey, Timeout: timeout}
}

// WithFlowbotAgent injects Flowbot URL and agent:headless token for flowbot-agent.
func (c *Cursor) WithFlowbotAgent(flowbotURL, agentToken string) *Cursor {
	c.FlowbotURL = flowbotURL
	c.AgentAccessToken = agentToken
	return c
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
	cmd.Env = append(os.Environ(), c.childEnv()...)
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

func (c *Cursor) childEnv() []string {
	var env []string
	if c.APIKey != "" {
		env = append(env, "CURSOR_API_KEY="+c.APIKey)
	}
	if c.FlowbotURL != "" {
		env = append(env, "FLOWBOT_URL="+c.FlowbotURL)
	}
	if c.AgentAccessToken != "" {
		env = append(env, "FLOWBOT_AGENT_TOKEN="+c.AgentAccessToken)
	}
	return env
}

func truncateForLog(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
