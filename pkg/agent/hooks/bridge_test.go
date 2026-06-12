package hooks_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridgeConfigComposesWithoutReplacingRouter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*hooks.Registry)
		base       msg.Config
		wantRouter bool
		wantHook   bool
	}{
		{
			name: "dual model router preserved with context hook",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
			},
			base:       msg.Config{ChatModel: "chat", ToolModel: "tool"},
			wantRouter: true,
			wantHook:   true,
		},
		{
			name:       "router only when registry empty",
			setup:      func(_ *hooks.Registry) {},
			base:       msg.Config{ChatModel: "chat", ToolModel: "tool"},
			wantRouter: true,
			wantHook:   false,
		},
		{
			name: "existing transform preserved",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
			},
			base: msg.Config{
				ChatModel: "chat",
				ToolModel: "tool",
				TransformContext: func(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
					return append(messages, userMessage("base")), nil
				},
			},
			wantRouter: true,
			wantHook:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			routed := model.ApplyDefaultRouter(tt.base)
			bridged := hooks.BridgeConfig(context.Background(), reg, routed)
			if tt.wantRouter {
				require.NotNil(t, bridged.PrepareNextTurn)
			}
			if tt.wantHook {
				require.NotNil(t, bridged.TransformContext)
			}
			if tt.base.TransformContext != nil && tt.wantHook {
				out, err := bridged.TransformContext([]msg.AgentMessage{userMessage("seed")})
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(out), 1)
			}
		})
	}
}

func TestBridgeConfigSkipsObserveOnlyRegistry(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*hooks.Registry)
		wantSame bool
	}{
		{
			name: "observe only does not wrap loop config",
			setup: func(reg *hooks.Registry) {
				hooks.Observe(reg, func(context.Context, hooks.ObservationEvent) error { return nil })
			},
			wantSame: true,
		},
		{
			name: "context hook wraps loop config",
			setup: func(reg *hooks.Registry) {
				hooks.OnContext(reg, func(_ context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
					return &hooks.ContextResult{Messages: ev.Messages}, nil
				})
			},
			wantSame: false,
		},
		{
			name:     "empty registry leaves config unchanged",
			setup:    func(_ *hooks.Registry) {},
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := hooks.NewRegistry()
			tt.setup(reg)
			base := msg.Config{ChatModel: "chat", ToolModel: "tool"}
			routed := model.ApplyDefaultRouter(base)
			bridged := hooks.BridgeConfig(context.Background(), reg, routed)
			if tt.wantSame {
				assert.False(t, reg.HasLoopHandlers())
				assert.Nil(t, bridged.TransformContext)
				return
			}
			require.NotNil(t, bridged.TransformContext)
		})
	}
}
