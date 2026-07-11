package chatagent

import (
	"context"

	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
	"github.com/flowline-io/flowbot/pkg/types"
)

var pipelineRunService = NewService()

// RunPipelineAgent executes one pipeline agent step via an ephemeral session.
func RunPipelineAgent(ctx context.Context, prompt string, uid types.Uid) (*abilityagent.RunResult, error) {
	out, err := RunEphemeral(ctx, pipelineRunService, EphemeralRunParams{
		UID:    uid,
		Prompt: prompt,
		Kind:   RunKindPipeline,
	})
	if err != nil {
		if out.SessionID != "" {
			return &abilityagent.RunResult{SessionID: out.SessionID}, err
		}
		return nil, err
	}
	return &abilityagent.RunResult{
		Reply:     out.Reply,
		SessionID: out.SessionID,
	}, nil
}

// PipelineAgentRunner implements abilityagent.Runner for pipeline steps.
type PipelineAgentRunner struct{}

// Run executes one pipeline agent step.
func (PipelineAgentRunner) Run(ctx context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	return RunPipelineAgent(ctx, params.Prompt, params.UID)
}
