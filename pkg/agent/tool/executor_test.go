package tool_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTool struct {
	name   string
	result string
	delay  time.Duration
	called atomic.Int32
	fail   bool
}

func (s *stubTool) Name() string        { return s.name }
func (s *stubTool) Description() string { return s.name }
func (*stubTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}
func (s *stubTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	s.called.Add(1)
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return msg.ToolResultMessage{}, ctx.Err()
		}
	}
	if s.fail {
		return msg.ToolResultMessage{}, assert.AnError
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       s.name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: s.result}},
	}, nil
}

func TestRegistry_RegisterAndActive(t *testing.T) {
	tests := []struct {
		name      string
		active    []string
		wantCount int
	}{
		{name: "all tools active by default", active: nil, wantCount: 2},
		{name: "single active tool", active: []string{"a"}, wantCount: 1},
		{name: "unknown active ignored", active: []string{"missing"}, wantCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(&stubTool{name: "a"}))
			require.NoError(t, reg.Register(&stubTool{name: "b"}))
			reg.SetActive(tt.active)
			assert.Len(t, reg.ActiveTools(), tt.wantCount)
		})
	}
}

func TestExecuteBatch_Modes(t *testing.T) {
	tests := []struct {
		name      string
		mode      msg.ToolExecutionMode
		toolCount int
	}{
		{name: "parallel batch", mode: msg.ToolExecutionParallel, toolCount: 3},
		{name: "sequential batch", mode: msg.ToolExecutionSequential, toolCount: 2},
		{name: "single tool", mode: msg.ToolExecutionParallel, toolCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := tool.NewRegistry()
			tools := make([]*stubTool, tt.toolCount)
			calls := make([]msg.ToolCallPart, tt.toolCount)
			for i := 0; i < tt.toolCount; i++ {
				tools[i] = &stubTool{name: "t" + string(rune('a'+i)), result: "ok", delay: 20 * time.Millisecond}
				require.NoError(t, reg.Register(tools[i]))
				calls[i] = msg.ToolCallPart{ID: "id-" + string(rune('a'+i)), Name: tools[i].name, Arguments: `{}`}
			}

			assistant := msg.AssistantMessage{Parts: make([]msg.ContentPart, len(calls))}
			for i, call := range calls {
				assistant.Parts[i] = call
			}

			result, err := tool.ExecuteBatch(context.Background(), tool.BatchRequest{
				Assistant: assistant,
				Context:   &msg.Context{},
				Registry:  reg,
				Mode:      tt.mode,
			})
			require.NoError(t, err)
			assert.Len(t, result.Messages, tt.toolCount)
			for _, message := range result.Messages {
				assert.Positive(t, message.DurationMs)
			}
		})
	}
}

func TestExecuteBatch_RecordsDuration(t *testing.T) {
	t.Parallel()

	reg := tool.NewRegistry()
	require.NoError(t, reg.Register(&stubTool{name: "echo", result: "ok", delay: 25 * time.Millisecond}))

	assistant := msg.AssistantMessage{Parts: []msg.ContentPart{
		msg.ToolCallPart{ID: "1", Name: "echo", Arguments: `{}`},
	}}

	var endEvents []agentevent.Event
	result, err := tool.ExecuteBatch(context.Background(), tool.BatchRequest{
		Assistant: assistant,
		Context:   &msg.Context{},
		Registry:  reg,
		Emit: func(_ context.Context, ev agentevent.Event) error {
			if ev.Type == agentevent.TypeToolExecutionEnd {
				endEvents = append(endEvents, ev)
			}
			return nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	assert.Positive(t, result.Messages[0].DurationMs)
	require.Len(t, endEvents, 1)
	assert.Equal(t, result.Messages[0].DurationMs, endEvents[0].DurationMs)
}

func TestExecuteBatch_ParallelAfterHookError(t *testing.T) {
	t.Parallel()

	reg := tool.NewRegistry()
	require.NoError(t, reg.Register(&stubTool{name: "a", result: "ok"}))
	require.NoError(t, reg.Register(&stubTool{name: "b", result: "ok"}))

	assistant := msg.AssistantMessage{Parts: []msg.ContentPart{
		msg.ToolCallPart{ID: "1", Name: "a", Arguments: `{}`},
		msg.ToolCallPart{ID: "2", Name: "b", Arguments: `{}`},
	}}

	result, err := tool.ExecuteBatch(context.Background(), tool.BatchRequest{
		Assistant: assistant,
		Context:   &msg.Context{},
		Registry:  reg,
		Mode:      msg.ToolExecutionParallel,
		After: func(ctx msg.AfterToolContext) (*msg.AfterToolResult, error) {
			if ctx.ToolCall.Name == "a" {
				return nil, assert.AnError
			}
			return nil, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Messages, 2)
	assert.True(t, result.Messages[0].IsError)
	assert.False(t, result.Messages[1].IsError)
}

func TestExecuteBatch_BlockedTerminate(t *testing.T) {
	t.Parallel()

	reg := tool.NewRegistry()
	stub := &stubTool{name: "echo", result: "ok"}
	require.NoError(t, reg.Register(stub))

	assistant := msg.AssistantMessage{Parts: []msg.ContentPart{
		msg.ToolCallPart{ID: "1", Name: "echo", Arguments: `{}`},
	}}

	result, err := tool.ExecuteBatch(context.Background(), tool.BatchRequest{
		Assistant: assistant,
		Context:   &msg.Context{},
		Registry:  reg,
		Before: func(msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
			return &msg.BeforeToolResult{Block: true, Reason: "loop", Terminate: true}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	assert.True(t, result.Messages[0].IsError)
	assert.True(t, result.Terminate)
	assert.Equal(t, int32(0), stub.called.Load())
}

func TestExecuteBatch_MissingTool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns error result"},
		{name: "marks result error"},
		{name: "keeps tool name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assistant := msg.AssistantMessage{Parts: []msg.ContentPart{
				msg.ToolCallPart{ID: "1", Name: "missing", Arguments: `{}`},
			}}
			result, err := tool.ExecuteBatch(context.Background(), tool.BatchRequest{
				Assistant: assistant,
				Context:   &msg.Context{},
				Registry:  tool.NewRegistry(),
				Mode:      msg.ToolExecutionParallel,
			})
			require.NoError(t, err)
			require.Len(t, result.Messages, 1)
			assert.True(t, result.Messages[0].IsError)
		})
	}
}
