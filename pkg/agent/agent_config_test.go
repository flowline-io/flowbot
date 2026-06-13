package agent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentApplyConfigPersistsAcrossPrompts(t *testing.T) {
	tests := []struct {
		name      string
		apply     func(*agent.Agent)
		wantSteps int
	}{
		{
			name: "updates max steps before prompt",
			apply: func(a *agent.Agent) {
				a.ApplyConfig(func(cfg *agent.Config) {
					cfg.MaxSteps = 1
				})
			},
			wantSteps: 1,
		},
		{
			name: "preserves steering queue drains",
			apply: func(a *agent.Agent) {
				a.ApplyConfig(func(cfg *agent.Config) {
					cfg.MaxSteps = 2
				})
			},
			wantSteps: 2,
		},
		{
			name: "config snapshot matches applied value",
			apply: func(a *agent.Agent) {
				a.ApplyConfig(func(cfg *agent.Config) {
					cfg.MaxSteps = 3
				})
			},
			wantSteps: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			ag := agent.NewAgent(agent.Options{Model: fakeModel, Config: agent.DefaultConfig()})
			tt.apply(ag)
			assert.Equal(t, tt.wantSteps, ag.Config().MaxSteps)

			stream, err := ag.Prompt(ctx, agent.NewUserMessage("hello"))
			require.NoError(t, err)
			_, err = stream.Await(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSteps, ag.Config().MaxSteps)
		})
	}
}

func TestAgentConsecutivePromptsAfterAwait(t *testing.T) {
	tests := []struct {
		name    string
		prompts int
	}{
		{name: "single prompt completes", prompts: 1},
		{name: "two consecutive prompts succeed", prompts: 2},
		{name: "three consecutive prompts succeed", prompts: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})
			ag := agent.NewAgent(agent.Options{Model: fakeModel, Config: agent.DefaultConfig()})

			for range tt.prompts {
				stream, err := ag.Prompt(ctx, agent.NewUserMessage("hello"))
				require.NoError(t, err)
				result, err := stream.Await(ctx)
				require.NoError(t, err)
				require.NoError(t, result.Err)
			}
			assert.Len(t, ag.State().Messages, tt.prompts*2)
		})
	}
}
