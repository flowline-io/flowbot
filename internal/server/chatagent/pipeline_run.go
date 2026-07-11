package chatagent

import (
	"context"
	"strings"
	"time"

	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

var pipelineRunService = NewService()

// RunPipelineAgent executes one pipeline agent step via an ephemeral session.
func RunPipelineAgent(ctx context.Context, prompt string, uid types.Uid) (*abilityagent.RunResult, error) {
	start := time.Now()
	flog.Info("[pipeline-agent] run start uid=%s prompt_len=%d run_timeout=%s ctx_deadline=%s",
		uid, len(strings.TrimSpace(prompt)), RunTimeout(), formatContextDeadline(ctx))

	out, err := RunEphemeral(ctx, pipelineRunService, EphemeralRunParams{
		UID:    uid,
		Prompt: prompt,
		Kind:   RunKindPipeline,
	})
	duration := time.Since(start).Round(time.Millisecond)
	if err != nil {
		flog.Info("[pipeline-agent] run failed uid=%s session=%s duration=%s err=%v",
			uid, out.SessionID, duration, err)
		if out.SessionID != "" {
			return &abilityagent.RunResult{SessionID: out.SessionID}, err
		}
		return nil, err
	}
	flog.Info("[pipeline-agent] run done uid=%s session=%s reply_len=%d duration=%s",
		uid, out.SessionID, len(out.Reply), duration)
	return &abilityagent.RunResult{
		Reply:     out.Reply,
		SessionID: out.SessionID,
	}, nil
}

func formatContextDeadline(ctx context.Context) string {
	if ctx == nil {
		return "none"
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return "none"
	}
	return time.Until(deadline).Round(time.Millisecond).String()
}

// PipelineAgentRunner implements abilityagent.Runner for pipeline steps.
type PipelineAgentRunner struct{}

// Run executes one pipeline agent step.
func (PipelineAgentRunner) Run(ctx context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	return RunPipelineAgent(ctx, params.Prompt, params.UID)
}
