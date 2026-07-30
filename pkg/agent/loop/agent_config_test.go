package loop_test

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"testing"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentApplyConfigPersistsAcrossPrompts(t *testing.T) {
	tests := []struct {
		name      string
		apply     func(*loop.Agent)
		wantSteps int
	}{
		{
			name: "updates max steps before prompt",
			apply: func(a *loop.Agent) {
				a.ApplyConfig(func(cfg *msg.Config) {
					cfg.MaxSteps = 1
				})
			},
			wantSteps: 1,
		},
		{
			name: "preserves steering queue drains",
			apply: func(a *loop.Agent) {
				a.ApplyConfig(func(cfg *msg.Config) {
					cfg.MaxSteps = 2
				})
			},
			wantSteps: 2,
		},
		{
			name: "config snapshot matches applied value",
			apply: func(a *loop.Agent) {
				a.ApplyConfig(func(cfg *msg.Config) {
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
			ag := loop.NewAgent(loop.Options{Model: fakeModel, Config: loop.DefaultConfig()})
			tt.apply(ag)
			assert.Equal(t, tt.wantSteps, ag.Config().MaxSteps)

			stream, err := ag.Prompt(ctx, loop.NewUserMessage("hello"))
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
			ag := loop.NewAgent(loop.Options{Model: fakeModel, Config: loop.DefaultConfig()})

			for range tt.prompts {
				stream, err := ag.Prompt(ctx, loop.NewUserMessage("hello"))
				require.NoError(t, err)
				result, err := stream.Await(ctx)
				require.NoError(t, err)
				require.NoError(t, result.Err)
			}
			assert.Len(t, ag.State().Messages, tt.prompts*2)
		})
	}
}
