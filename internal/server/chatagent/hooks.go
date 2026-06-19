package chatagent

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ChatHookDeps carries per-run metadata for chat agent hook handlers.
type ChatHookDeps struct {
	SessionID   string
	UID         types.Uid
	SessionMode string
}

// RegisterHooks wires observational and API hooks for one chat agent harness run.
func RegisterHooks(reg *hooks.Registry, deps ChatHookDeps) {
	if reg == nil {
		return
	}

	registerPermissionHook(reg, deps)

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

func registerPermissionHook(reg *hooks.Registry, deps ChatHookDeps) {
	hooks.OnToolCall(reg, func(ctx context.Context, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		uid := deps.UID
		if uid.IsZero() {
			var err error
			uid, err = SessionOwnerUID(ctx, deps.SessionID)
			if err != nil {
				return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, nil
			}
		}

		cfg, err := LoadUserPermissions(ctx, uid)
		if err != nil {
			return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, nil
		}
		evaluator := permission.NewEvaluator(cfg)
		sessionState := permissionSessions.GetPermissionSession(deps.SessionID)
		workspaceRoot := config.App.ChatAgent.Workspace
		externalPath := detectExternalPath(event, workspaceRoot)

		if block := planModeToolBlock(deps, event.ToolCall.Name); block != nil {
			return block, nil
		}

		result := evaluator.Evaluate(permission.Request{
			Tool:          event.ToolCall.Name,
			Args:          event.Args,
			WorkspaceRoot: workspaceRoot,
			ExternalPath:  externalPath,
		}, sessionState)

		return evaluatePermissionResult(ctx, deps.SessionID, event, result, sessionState)
	})
}

func evaluatePermissionResult(
	ctx context.Context,
	sessionID string,
	event hooks.ToolCallEvent,
	result permission.Result,
	sessionState *permission.SessionState,
) (*hooks.ToolCallResult, error) {
	switch result.Action {
	case permission.ActionAllow:
		return nil, nil
	case permission.ActionDeny:
		return &hooks.ToolCallResult{Block: true, Reason: "permission denied"}, nil
	case permission.ActionAsk:
		return handlePermissionAsk(ctx, sessionID, event, result, sessionState)
	default:
		return &hooks.ToolCallResult{Block: true, Reason: "permission denied"}, nil
	}
}

func handlePermissionAsk(
	ctx context.Context,
	sessionID string,
	event hooks.ToolCallEvent,
	result permission.Result,
	sessionState *permission.SessionState,
) (*hooks.ToolCallResult, error) {
	raw, ok := sessionConfirmGates.Load(sessionID)
	if !ok {
		flog.Debug("[chat-agent] ask allowed without confirm gate session=%s tool=%s",
			sessionID, event.ToolCall.Name)
		return nil, nil
	}
	gate, ok := raw.(*ConfirmGate)
	if !ok {
		return &hooks.ToolCallResult{Block: true, Reason: "approval required"}, nil
	}
	resp, err := gate.Wait(ctx, event, result)
	if err != nil {
		return &hooks.ToolCallResult{Block: true, Reason: err.Error()}, nil
	}
	if !resp.Approved {
		return &hooks.ToolCallResult{Block: true, Reason: "user denied"}, nil
	}
	if resp.Mode == ConfirmModeAlways {
		pattern, grantOK := alwaysGrantPattern(result, resp.Pattern)
		if !grantOK {
			flog.Warn("[chat-agent] always grant rejected session=%s key=%s", sessionID, result.PermissionKey)
		} else if err := sessionState.AddGrant(result.PermissionKey, pattern); err != nil {
			flog.Warn("[chat-agent] always grant rejected session=%s: %v", sessionID, err)
		}
	}
	return nil, nil
}

func detectExternalPath(event hooks.ToolCallEvent, workspaceRoot string) bool {
	switch event.ToolCall.Name {
	case permission.ToolReadFile, permission.ToolWriteFile:
		path := fmt.Sprint(event.Args["path"])
		ws := coding.Workspace{Root: workspaceRoot}
		if !ws.ResolvePath(path).IsOk() {
			return true
		}
	}
	return false
}

func planModeToolBlock(deps ChatHookDeps, toolName string) *hooks.ToolCallResult {
	mode := deps.SessionMode
	if mode == "" {
		mode = ModeNormal
	}
	if mode != ModePlan || IsReadOnlyTool(toolName) {
		return nil
	}
	return &hooks.ToolCallResult{Block: true, Reason: "plan mode: read-only"}
}
