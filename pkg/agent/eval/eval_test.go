package eval_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/eval"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		scenario          eval.Scenario
		wantToolSelection bool
		wantArgsValid     bool
		wantCompleted     bool
		wantMinSteps      int
	}{
		{
			name: "correct tool and args then complete",
			scenario: eval.Scenario{
				Name:   "echo happy path",
				Prompt: "echo hi",
				Tools:  []tool.Tool{echo.Tool{}},
				Scripts: []agentllm.ResponseScript{
					eval.ToolCallScript("c1", "echo", `{"text":"hi"}`),
					eval.TextScript("done"),
				},
				Expect: eval.Expectation{
					ExpectedTools:     []string{"echo"},
					RequiredArgs:      map[string][]string{"echo": {"text"}},
					MaxSteps:          5,
					RequireCompletion: true,
				},
			},
			wantToolSelection: true,
			wantArgsValid:     true,
			wantCompleted:     true,
			wantMinSteps:      2,
		},
		{
			name: "wrong tool selection",
			scenario: eval.Scenario{
				Name:   "wrong tool",
				Prompt: "echo hi",
				Tools:  []tool.Tool{echo.Tool{}},
				Scripts: []agentllm.ResponseScript{
					eval.TextScript("I will not call tools"),
				},
				Expect: eval.Expectation{
					ExpectedTools:     []string{"echo"},
					RequireCompletion: true,
				},
			},
			wantToolSelection: false,
			wantArgsValid:     true,
			wantCompleted:     true,
			wantMinSteps:      1,
		},
		{
			name: "invalid args scored false",
			scenario: eval.Scenario{
				Name:   "missing arg",
				Prompt: "echo",
				Tools:  []tool.Tool{echo.Tool{}},
				Scripts: []agentllm.ResponseScript{
					eval.ToolCallScript("c1", "echo", `{"text":""}`),
					eval.TextScript("done"),
				},
				Expect: eval.Expectation{
					ExpectedTools:     []string{"echo"},
					RequiredArgs:      map[string][]string{"echo": {"text"}},
					RequireCompletion: true,
				},
			},
			wantToolSelection: true,
			wantArgsValid:     false,
			wantCompleted:     true,
			wantMinSteps:      2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(tt.scenario.Scripts...)
			reg := tool.NewRegistry()
			for _, item := range tt.scenario.Tools {
				require.NoError(t, reg.Register(item))
			}
			cfg := agent.DefaultConfig()
			cfg.ModelName = "eval-fake"
			if tt.scenario.Expect.MaxSteps > 0 {
				cfg.MaxSteps = tt.scenario.Expect.MaxSteps
			}
			messages, err := agent.RunLoop(context.Background(), []agent.AgentMessage{
				agent.NewUserMessage(tt.scenario.Prompt),
			}, &agent.Context{}, cfg, agent.LoopDeps{Model: model, Registry: reg}, nil)
			metrics := eval.Score(messages, tt.scenario.Expect, err)
			assert.Equal(t, tt.wantToolSelection, metrics.ToolSelectionCorrect)
			assert.Equal(t, tt.wantArgsValid, metrics.ArgsValid)
			assert.Equal(t, tt.wantCompleted, metrics.Completed)
			assert.GreaterOrEqual(t, metrics.StepCount, tt.wantMinSteps)
		})
	}
}
