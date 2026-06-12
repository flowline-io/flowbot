package chatagent

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// ChatHookDeps carries per-run metadata for chat agent hook handlers.
type ChatHookDeps struct {
	SessionID string
}

// RegisterHooks wires observational and API hooks for one chat agent harness run.
func RegisterHooks(reg *hooks.Registry, deps ChatHookDeps) {
	if reg == nil {
		return
	}

	registerConfirmHook(reg, deps.SessionID)

	hooks.Observe(reg, func(_ context.Context, event hooks.ObservationEvent) error {
		switch event.Type {
		case hooks.EventContextUsage:
			if event.ContextUsage == nil {
				return nil
			}
			flog.Debug("[chat-agent] context usage session=%s tokens=%d window=%d percent=%.1f",
				deps.SessionID,
				event.ContextUsage.Tokens,
				event.ContextUsage.ContextWindow,
				event.ContextUsage.Percent,
			)
			if publisher := activePublisher(deps.SessionID); publisher != nil {
				PublishUsageEvent(
					publisher,
					0,
					0,
					event.ContextUsage.Tokens,
					event.ContextUsage.ContextWindow,
					event.ContextUsage.Percent,
				)
			}
		case hooks.EventSavePoint:
			flog.Debug("[chat-agent] save_point session=%s", deps.SessionID)
		}
		return nil
	})
}

func activePublisher(sessionID string) EventPublisher {
	raw, ok := activeAPIRuns.Load(sessionID)
	if !ok {
		return nil
	}
	state, ok := raw.(*APIRunState)
	if !ok || state.publisher == nil {
		return nil
	}
	return state.publisher
}

func registerConfirmHook(reg *hooks.Registry, sessionID string) {
	hooks.OnToolCall(reg, func(ctx context.Context, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		if !toolNeedsConfirm(event.ToolCall.Name) {
			return nil, nil
		}
		raw, ok := sessionConfirmGates.Load(sessionID)
		if !ok {
			return nil, nil
		}
		gate, ok := raw.(*ConfirmGate)
		if !ok {
			return nil, nil
		}
		approved, err := gate.Wait(ctx, event)
		if err != nil {
			return &hooks.ToolCallResult{Block: true, Reason: err.Error()}, nil
		}
		if !approved {
			return &hooks.ToolCallResult{Block: true, Reason: "user denied"}, nil
		}
		return nil, nil
	})
}
