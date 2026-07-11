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
func RunPipelineAgent(ctx context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	if err := validatePipelineAgentTools(params.Tools); err != nil {
		return nil, err
	}

	prompt := strings.TrimSpace(params.Prompt)
	start := time.Now()
	flog.Info("[pipeline-agent] run start uid=%s prompt_len=%d tools=%d skills=%d run_timeout=%s ctx_deadline=%s",
		params.UID, len(prompt), len(params.Tools), len(params.Skills), RunTimeout(), formatContextDeadline(ctx))

	out, err := RunEphemeral(ctx, pipelineRunService, EphemeralRunParams{
		UID:    params.UID,
		Prompt: prompt,
		Kind:   RunKindPipeline,
		Tools:  params.Tools,
		Skills: params.Skills,
	})
	duration := time.Since(start).Round(time.Millisecond)
	if err != nil {
		flog.Info("[pipeline-agent] run failed uid=%s session=%s duration=%s err=%v",
			params.UID, out.SessionID, duration, err)
		if out.SessionID != "" {
			return &abilityagent.RunResult{SessionID: out.SessionID}, err
		}
		return nil, err
	}
	flog.Info("[pipeline-agent] run done uid=%s session=%s reply_len=%d duration=%s",
		params.UID, out.SessionID, len(out.Reply), duration)
	return &abilityagent.RunResult{
		Reply:     out.Reply,
		SessionID: out.SessionID,
	}, nil
}

func validatePipelineAgentTools(tools []string) error {
	if len(tools) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(SelectableSubagentTools()))
	for _, name := range SelectableSubagentTools() {
		allowed[name] = struct{}{}
	}
	for _, toolName := range tools {
		if _, ok := allowed[toolName]; !ok {
			return types.Errorf(types.ErrInvalidArgument, "tool %s is not allowed", toolName)
		}
	}
	return nil
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
	return RunPipelineAgent(ctx, params)
}
