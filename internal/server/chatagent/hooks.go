package chatagent

import (
	"context"
	"fmt"

	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ReasonConfirmRequiredPlatform is returned when ActionAsk cannot be resolved without a ConfirmGate.
const ReasonConfirmRequiredPlatform = "This action requires approval. " +
	"Use the Web UI or configure permissions via PUT /chatagent/permissions."

// ChatHookDeps carries per-run metadata for chat agent hook handlers.
type ChatHookDeps struct {
	SessionID   string
	UID         types.Uid
	SessionMode string
	Kind        RunKind
	// DCG is the pre-permission command guard. Nil uses dcg.DefaultChecker().
	DCG dcg.Checker
}

// RegisterHooks wires observational and API hooks for one chat agent harness run.
func RegisterHooks(reg *hooks.Registry, deps ChatHookDeps) {
	if reg == nil {
		return
	}

	registerDCGHook(reg, deps)
	registerPermissionHook(reg, deps)
	registerPathSensors(reg)
	registerLintSensor(reg)
	registerProgressHooks(reg)

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

func registerDCGHook(reg *hooks.Registry, deps ChatHookDeps) {
	hooks.OnToolCall(reg, func(ctx context.Context, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		command, ok, err := dcg.CommandForTool(event.ToolCall.Name, event.Args)
		if err != nil {
			flog.Warn("[chat-agent] dcg synth failed session=%s tool=%s: %v",
				deps.SessionID, event.ToolCall.Name, err)
			return &hooks.ToolCallResult{Block: true, Reason: err.Error()}, nil
		}
		if !ok {
			return nil, nil
		}
		flog.Debug("[chat-agent] dcg check session=%s tool=%s command=%q",
			deps.SessionID, event.ToolCall.Name, dcg.TruncateCommandForLog(command))
		checker := deps.DCG
		if checker == nil {
			checker = dcg.DefaultChecker()
		}
		decision, err := checker.Check(ctx, command)
		if err != nil {
			flog.Warn("[chat-agent] dcg check error session=%s tool=%s command=%q: %v",
				deps.SessionID, event.ToolCall.Name, dcg.TruncateCommandForLog(command), err)
			return &hooks.ToolCallResult{Block: true, Reason: err.Error()}, nil
		}
		if !decision.Allow {
			reason := decision.Reason
			if reason == "" {
				reason = dcg.ReasonBlocked
			}
			flog.Info("[chat-agent] dcg blocked session=%s tool=%s rule=%s pack=%s reason=%s command=%q",
				deps.SessionID, event.ToolCall.Name, decision.RuleID, decision.PackID, reason, dcg.TruncateCommandForLog(command))
			return &hooks.ToolCallResult{Block: true, Reason: reason}, nil
		}
		flog.Debug("[chat-agent] dcg allowed session=%s tool=%s command=%q",
			deps.SessionID, event.ToolCall.Name, dcg.TruncateCommandForLog(command))
		return nil, nil
	})
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
		if IsAutonomousRunKind(deps.Kind) {
			cfg = permission.Merge(cfg, permission.ScheduledRunOverlay())
		}
		evaluator := permission.NewEvaluator(cfg)
		sessionState := permissionSessions.GetPermissionSession(ctx, deps.SessionID)
		workspaceRoot := config.App.ChatAgent.Workspace
		externalPath := detectExternalPath(event, workspaceRoot)

		if block := planModeToolBlock(ctx, deps.SessionID, event); block != nil {
			return block, nil
		}

		result := evaluator.Evaluate(permission.Request{
			Tool:          event.ToolCall.Name,
			Args:          event.Args,
			WorkspaceRoot: workspaceRoot,
			ExternalPath:  externalPath,
		}, sessionState)

		if result.DoomLoopTriggered {
			metrics.Agent().IncDoomLoop(event.ToolCall.Name)
		}

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
		flog.Debug("[chat-agent] ask blocked without confirm gate session=%s tool=%s",
			sessionID, event.ToolCall.Name)
		return &hooks.ToolCallResult{Block: true, Reason: ReasonConfirmRequiredPlatform}, nil
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
		} else {
			PersistSessionGrants(ctx, sessionID, sessionState)
		}
	}
	return nil, nil
}

func detectExternalPath(event hooks.ToolCallEvent, workspaceRoot string) bool {
	ws := coding.Workspace{Root: workspaceRoot}
	switch event.ToolCall.Name {
	case permission.ToolReadFile, permission.ToolWriteFile,
		permission.ToolListDir, permission.ToolGlobFiles, permission.ToolGrepFiles:
		path := strings.TrimSpace(fmt.Sprint(event.Args["path"]))
		if path == "" || path == "<nil>" {
			return false
		}
		return !ws.ResolvePath(path).IsOk()
	case permission.ToolApplyPatch:
		for _, path := range coding.PatchFilePaths(fmt.Sprint(event.Args["patch"])) {
			if !ws.ResolvePath(path).IsOk() {
				return true
			}
		}
	}
	return false
}

func planModeToolBlock(ctx context.Context, sessionID string, event hooks.ToolCallEvent) *hooks.ToolCallResult {
	if LoadSessionMode(ctx, sessionID) != ModePlan {
		return nil
	}
	toolName := event.ToolCall.Name
	switch toolName {
	case memorySetToolName, memoryDeleteToolName:
		return &hooks.ToolCallResult{Block: true, Reason: "plan mode: memory write disabled"}
	}
	if IsReadOnlyTool(toolName) {
		return nil
	}
	reason := "plan mode: read-only"
	if IsScheduleWriteTool(toolName) {
		reason = "plan mode: schedule write tools are disabled"
	}
	return &hooks.ToolCallResult{Block: true, Reason: reason}
}
