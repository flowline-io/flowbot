package loop_test

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"testing"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLoop_StreamingEvents(t *testing.T) {
	tests := []struct {
		name         string
		chunks       []string
		wantUpdates  int
		wantComplete string
	}{
		{name: "emits start update end", chunks: []string{"hel", "lo"}, wantUpdates: 2, wantComplete: "hello"},
		{name: "single chunk", chunks: []string{"done"}, wantUpdates: 1, wantComplete: "done"},
		{name: "many chunks", chunks: []string{"a", "b", "c"}, wantUpdates: 3, wantComplete: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Chunks: tt.chunks})
			stream := agentevent.NewStream(32)

			var events []agentevent.Event
			stream.Subscribe(func(ev agentevent.Event) error {
				events = append(events, ev)
				return nil
			})

			cfg := loop.DefaultConfig()
			cfg.MaxSteps = 5
			cfg.ModelName = "fake"

			_, err := loop.RunLoop(context.Background(), []msg.AgentMessage{
				loop.NewUserMessage("stream"),
			}, &msg.Context{}, cfg, loop.LoopDeps{Model: model}, stream)
			require.NoError(t, err)

			var starts, updates, ends int
			for _, ev := range events {
				switch ev.Type {
				case agentevent.TypeMessageStart:
					if _, ok := ev.Message.(msg.AssistantMessage); ok {
						starts++
					}
				case agentevent.TypeMessageUpdate:
					updates++
				case agentevent.TypeMessageEnd:
					if _, ok := ev.Message.(msg.AssistantMessage); ok {
						ends++
					}
				}
			}
			assert.Equal(t, 1, starts)
			assert.Equal(t, tt.wantUpdates, updates)
			assert.Equal(t, 1, ends)
		})
	}
}

func TestRunLoop_ReasoningStream(t *testing.T) {
	tests := []struct {
		name          string
		reasoning     []string
		wantReasoning bool
	}{
		{name: "emits reasoning deltas", reasoning: []string{"think", "ing"}, wantReasoning: true},
		{name: "empty reasoning skipped", reasoning: nil, wantReasoning: false},
		{name: "single reasoning chunk", reasoning: []string{"plan"}, wantReasoning: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{
				ReasoningChunks: tt.reasoning,
				Content:         "answer",
			})
			stream := agentevent.NewStream(32)
			var reasoningUpdates int
			stream.Subscribe(func(ev agentevent.Event) error {
				if ev.ReasoningDelta != "" {
					reasoningUpdates++
				}
				return nil
			})

			cfg := loop.DefaultConfig()
			cfg.ModelName = "deepseek-v4-chat"
			cfg.MaxSteps = 3

			_, err := loop.RunLoop(context.Background(), []msg.AgentMessage{
				loop.NewUserMessage("reason"),
			}, &msg.Context{}, cfg, loop.LoopDeps{Model: model}, stream)
			require.NoError(t, err)

			if tt.wantReasoning {
				assert.Positive(t, reasoningUpdates)
				return
			}
			assert.Zero(t, reasoningUpdates)
		})
	}
}

func TestRunLoop_StreamingCancelled(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancel aborts streaming"},
		{name: "cancel returns aborted"},
		{name: "cancel stops loop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			stream := agentevent.NewStream(8)
			_, err := loop.RunLoop(ctx, []msg.AgentMessage{loop.NewUserMessage("x")}, &msg.Context{}, loop.DefaultConfig(), loop.LoopDeps{
				Model: agentllm.NewFakeModel(agentllm.ResponseScript{Chunks: []string{"x"}}),
			}, stream)
			assert.Error(t, err)
		})
	}
}
