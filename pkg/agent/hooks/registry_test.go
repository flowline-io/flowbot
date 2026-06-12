package hooks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitContextChainsReplacements(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*hooks.Registry)
		input   []msg.AgentMessage
		wantLen int
	}{
		{
			name: "single handler replaces messages",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(_ context.Context, _ hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: []msg.AgentMessage{userMessage("replaced")}}, nil
				})
			},
			input:   []msg.AgentMessage{userMessage("original")},
			wantLen: 1,
		},
		{
			name: "chained handlers apply in order",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: append(ev.Messages, userMessage("second"))}, nil
				})
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: append(ev.Messages, userMessage("third"))}, nil
				})
			},
			input:   []msg.AgentMessage{userMessage("first")},
			wantLen: 3,
		},
		{
			name:    "empty registry returns input",
			setup:   func(_ *hooks.Registry) {},
			input:   []msg.AgentMessage{userMessage("only")},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			got, err := reg.EmitContext(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestEmitToolCallStopsOnBlock(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*hooks.Registry)
		wantBlock bool
		wantErr   bool
	}{
		{
			name: "first handler blocks",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolCall(reg, func(_ context.Context, _ hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
					return &hooks.ToolCallResult{Block: true, Reason: "denied"}, nil
				})
				hooks.OnToolCall(reg, func(_ context.Context, _ hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
					return &hooks.ToolCallResult{Block: true, Reason: "later"}, nil
				})
			},
			wantBlock: true,
		},
		{
			name:      "no handlers does not block",
			setup:     func(_ *hooks.Registry) {},
			wantBlock: false,
		},
		{
			name: "handler error propagates",
			setup: func(reg *hooks.Registry) {
				hooks.OnToolCall(reg, func(_ context.Context, _ hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
					return nil, errors.New("boom")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{ID: "1", Name: "echo"},
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantBlock {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				assert.Equal(t, "denied", result.Reason)
				return
			}
			assert.Nil(t, result)
		})
	}
}

func TestEmitObservationContinuesOnHandlerError(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*hooks.Registry)
		wantWarns int
	}{
		{
			name: "failed observer does not stop siblings",
			setup: func(reg *hooks.Registry) {
				hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error {
					return errors.New("first")
				})
				hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error {
					return nil
				})
			},
			wantWarns: 1,
		},
		{
			name: "successful observers do not warn",
			setup: func(reg *hooks.Registry) {
				hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error { return nil })
			},
			wantWarns: 0,
		},
		{
			name:      "empty registry is no-op",
			setup:     func(_ *hooks.Registry) {},
			wantWarns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			warns := 0
			reg.EmitObservation(context.Background(), hooks.ObservationEvent{Type: hooks.EventSavePoint}, func(string, ...any) {
				warns++
			})
			assert.Equal(t, tt.wantWarns, warns)
		})
	}
}

func TestHasLoopHandlersIgnoresObservers(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*hooks.Registry)
		wantLoop  bool
		wantTotal bool
	}{
		{
			name: "observe only is not a loop handler",
			setup: func(reg *hooks.Registry) {
				hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error { return nil })
			},
			wantLoop:  false,
			wantTotal: true,
		},
		{
			name: "context hook is a loop handler",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(context.Context, hooks.ContextEvent) (*hooks.ContextResult, error) {
					return nil, nil
				})
			},
			wantLoop:  true,
			wantTotal: true,
		},
		{
			name:      "empty registry",
			setup:     func(_ *hooks.Registry) {},
			wantLoop:  false,
			wantTotal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			assert.Equal(t, tt.wantLoop, reg.HasLoopHandlers())
			assert.Equal(t, tt.wantTotal, reg.HasHandlers())
		})
	}
}

func userMessage(text string) msg.AgentMessage {
	return msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: text}}}
}
