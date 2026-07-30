package loop_test

import (
	"context"
	"sync"
	"testing"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

type stallTool struct{}

func (stallTool) Name() string        { return "stall" }
func (stallTool) Description() string { return "blocks until cancelled" }
func (stallTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (stallTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	<-ctx.Done()
	return msg.ToolResultMessage{ToolCallID: id, IsError: true}, nil
}

func toolCallModel() *agentllm.FakeModel {
	return agentllm.NewFakeModel(agentllm.ResponseScript{
		ToolCalls: []llms.ToolCall{{
			ID: "call-1", Type: "function",
			FunctionCall: &llms.FunctionCall{Name: "stall", Arguments: `{}`},
		}},
	})
}

func stallRegistry(t *testing.T) *tool.Registry {
	t.Helper()
	reg := tool.NewRegistry()
	require.NoError(t, reg.Register(stallTool{}))
	return reg
}

func TestNewUserMessageWithParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []msg.ContentPart
		want  int
	}{
		{name: "text part", parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}, want: 1},
		{name: "multiple parts", parts: []msg.ContentPart{
			msg.TextPart{Text: "a"},
			msg.TextPart{Text: "b"},
		}, want: 2},
		{name: "empty parts slice", parts: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := loop.NewUserMessageWithParts(tt.parts...)
			assert.Len(t, msg.Parts, tt.want)
			assert.False(t, msg.Timestamp.IsZero())
		})
	}
}

func TestAgentSteerAndFollowUpQueues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       msg.QueueMode
		steer      bool
		wantFirst  string
		wantRemain int
	}{
		{name: "steer queue all drains all", mode: msg.QueueAll, steer: true, wantFirst: "first", wantRemain: 0},
		{name: "steer queue one drains one", mode: msg.QueueOne, steer: true, wantFirst: "first", wantRemain: 1},
		{name: "follow up queue all", mode: msg.QueueAll, steer: false, wantFirst: "follow", wantRemain: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loop.DefaultConfig()
			if tt.steer {
				cfg.SteeringMode = tt.mode
			} else {
				cfg.FollowUpMode = tt.mode
			}
			ag := loop.NewAgent(loop.Options{Config: cfg})

			if tt.steer {
				ag.Steer(loop.NewUserMessage("first"))
				ag.Steer(loop.NewUserMessage("second"))
				msgs, err := ag.Config().GetSteeringMessages()
				require.NoError(t, err)
				require.NotEmpty(t, msgs)
				assert.Equal(t, tt.wantFirst, msgs[0].(msg.UserMessage).Parts[0].(msg.TextPart).Text)
				if tt.wantRemain > 0 {
					remaining, err := ag.Config().GetSteeringMessages()
					require.NoError(t, err)
					assert.Len(t, remaining, tt.wantRemain)
				}
				return
			}

			ag.FollowUp(loop.NewUserMessage("follow"))
			msgs, err := ag.Config().GetFollowUpMessages()
			require.NoError(t, err)
			require.Len(t, msgs, 1)
		})
	}
}

func TestAgentApplyStateAndSetTools(t *testing.T) {
	t.Parallel()

	reg := tool.NewRegistry()
	require.NoError(t, reg.Register(stallTool{}))

	ag := loop.NewAgent(loop.Options{})
	ag.ApplyState(func(state *msg.Context) {
		state.SystemPrompt = "updated"
		state.ModelName = "test-model"
	})
	state := ag.State()
	assert.Equal(t, "updated", state.SystemPrompt)
	assert.Equal(t, "test-model", state.ModelName)

	ag.SetTools(reg)
	ag.SetActiveTools([]string{"stall"})
	cfg := ag.Config()
	assert.NotNil(t, cfg)
}

func TestAgentSubscribeAndContinue(t *testing.T) {
	t.Parallel()

	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "continued"})
	ag := loop.NewAgent(loop.Options{
		Model: model,
		Config: msg.Config{
			ModelName: "fake",
			MaxSteps:  5,
		},
		InitialState: &msg.Context{
			Messages: []msg.AgentMessage{loop.NewUserMessage("seed")},
		},
	})

	var mu sync.Mutex
	var starts int
	ag.Subscribe(func(ev agentevent.Event) error {
		mu.Lock()
		defer mu.Unlock()
		if ev.Type == agentevent.TypeAgentStart {
			starts++
		}
		return nil
	})

	stream, err := ag.Continue(context.Background())
	require.NoError(t, err)
	result, err := stream.Await(context.Background())
	require.NoError(t, err)
	require.NoError(t, result.Err)

	mu.Lock()
	assert.GreaterOrEqual(t, starts, 1)
	mu.Unlock()
}

func TestAgentAbort(t *testing.T) {
	t.Parallel()

	ag := loop.NewAgent(loop.Options{
		Model:    toolCallModel(),
		Registry: stallRegistry(t),
		Config:   msg.Config{ModelName: "fake", MaxSteps: 5},
	})

	stream, err := ag.Prompt(context.Background(), loop.NewUserMessage("run"))
	require.NoError(t, err)

	time.AfterFunc(50*time.Millisecond, ag.Abort)

	result, err := stream.Await(context.Background())
	require.NoError(t, err)
	assert.ErrorIs(t, result.Err, loop.ErrAborted)
}

func TestAgentPromptWhileRunning(t *testing.T) {
	t.Parallel()

	ag := loop.NewAgent(loop.Options{
		Model:    toolCallModel(),
		Registry: stallRegistry(t),
		Config:   msg.Config{ModelName: "fake", MaxSteps: 5},
	})

	stream, err := ag.Prompt(context.Background(), loop.NewUserMessage("run"))
	require.NoError(t, err)

	time.AfterFunc(20*time.Millisecond, func() {
		_, secondErr := ag.Prompt(context.Background(), loop.NewUserMessage("again"))
		require.ErrorIs(t, secondErr, loop.ErrAborted)
		ag.Abort()
	})

	_, err = stream.Await(context.Background())
	require.NoError(t, err)
}

func TestAgentSetModel(t *testing.T) {
	t.Parallel()

	ag := loop.NewAgent(loop.Options{
		Model: agentllm.NewFakeModel(agentllm.ResponseScript{Content: "first"}),
	})
	ag.SetModel(agentllm.NewFakeModel(agentllm.ResponseScript{Content: "second"}))

	stream, err := ag.Prompt(context.Background(), loop.NewUserMessage("hi"))
	require.NoError(t, err)
	result, err := stream.Await(context.Background())
	require.NoError(t, err)
	require.NoError(t, result.Err)
}
