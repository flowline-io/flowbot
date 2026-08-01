package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
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
	// Service owns hot-path session state (permission sessions). Required for permission grants.
	Service *Service
	// Publisher is the SSE publisher when known at harness build time.
	// Prefer run-context injection via withRunIO for pooled harness turns.
	Publisher EventPublisher
	// Confirm is the active confirm gate when known at harness build time.
	// Prefer run-context injection via withRunIO for pooled harness turns.
	Confirm *ConfirmGate
	// DCG is the pre-permission command guard. Nil uses dcg.DefaultChecker().
	DCG dcg.Checker
	// Reviewer is the aux security reviewer for auto mode. Nil builds one lazily.
	Reviewer approval.Reviewer
	// Breaker is the per-run denial circuit breaker for auto mode. Nil creates a new one.
	Breaker *approval.Breaker
	// ApprovalMode overrides DB/YAML mode when Valid (tests and explicit wiring).
	ApprovalMode approval.Mode
}

// RegisterHooks wires observational and API hooks for one chat agent harness run.
func RegisterHooks(reg *hooks.Registry, deps ChatHookDeps) {
	if reg == nil {
		return
	}
	if deps.Breaker == nil {
		deps.Breaker = approval.NewBreaker(approvalDenialThreshold())
	}

	registerDCGHook(reg, deps)
	registerPermissionHook(reg, deps)
	registerPathSensors(reg)
	registerLintSensor(reg)
	registerProgressHooks(reg)

	hooks.Observe(reg, func(ctx context.Context, event hooks.ObservationEvent) error {
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
			publisher := resolveRunPublisher(ctx, deps)
			if publisher != nil {
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

func resolveRunPublisher(ctx context.Context, deps ChatHookDeps) EventPublisher {
	if io := runIOFromContext(ctx); io != nil && io.Publisher != nil {
		return io.Publisher
	}
	return deps.Publisher
}

func resolveRunConfirm(ctx context.Context, deps ChatHookDeps) *ConfirmGate {
	if io := runIOFromContext(ctx); io != nil && io.Confirm != nil {
		return io.Confirm
	}
	return deps.Confirm
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
		if block := planModeToolBlock(ctx, deps.SessionID, event); block != nil {
			return block, nil
		}
		if deps.Breaker != nil && deps.Breaker.Tripped() {
			return &hooks.ToolCallResult{Block: true, Reason: approval.ReasonBreakerTripped}, nil
		}

		uid := deps.UID
		if uid.IsZero() {
			var err error
			uid, err = SessionOwnerUID(ctx, deps.SessionID)
			if err != nil {
				return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, nil
			}
		}

		if IsAutonomousRunKind(deps.Kind) {
			return handleManualPermission(ctx, deps, uid, event, true)
		}

		mode := deps.ApprovalMode
		if !mode.Valid() {
			var loadErr error
			mode, loadErr = ResolveRunApprovalMode(ctx, deps.SessionID, uid)
			if loadErr != nil {
				return &hooks.ToolCallResult{Block: true, Reason: "approval mode unavailable"}, nil
			}
		}
		switch mode {
		case approval.ModeOff:
			return handleOffApproval(ctx, uid, event)
		case approval.ModeAuto:
			return handleAutoApproval(ctx, deps, uid, event)
		default:
			return handleManualPermission(ctx, deps, uid, event, false)
		}
	})
}

func handleManualPermission(
	ctx context.Context,
	deps ChatHookDeps,
	uid types.Uid,
	event hooks.ToolCallEvent,
	autonomous bool,
) (*hooks.ToolCallResult, error) {
	cfg, err := LoadUserPermissions(ctx, uid)
	if err != nil {
		return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, nil
	}
	if autonomous {
		cfg = permission.Merge(cfg, permission.ScheduledRunOverlay())
	}
	evaluator := permission.NewEvaluator(cfg)
	workspaceRoot := config.App.ChatAgent.Workspace
	externalPath := detectExternalPath(event, workspaceRoot)

	if deps.Service == nil {
		return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, nil
	}
	sessionState := deps.Service.permissionSessions.GetPermissionSession(ctx, deps.SessionID)

	result := evaluator.Evaluate(permission.Request{
		Tool:          event.ToolCall.Name,
		Args:          event.Args,
		WorkspaceRoot: workspaceRoot,
		ExternalPath:  externalPath,
	}, sessionState)

	if result.DoomLoopTriggered {
		metrics.Agent().IncDoomLoop(event.ToolCall.Name)
	}

	return evaluatePermissionResult(ctx, deps, event, result, sessionState)
}

func handleOffApproval(ctx context.Context, uid types.Uid, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
	block, _, err := evaluateDenyOnlyGate(ctx, uid, event)
	return block, err
}

func handleAutoApproval(
	ctx context.Context,
	deps ChatHookDeps,
	uid types.Uid,
	event hooks.ToolCallEvent,
) (*hooks.ToolCallResult, error) {
	block, req, err := evaluateDenyOnlyGate(ctx, uid, event)
	if err != nil || block != nil {
		return block, err
	}
	if approval.IsReadonlyTool(event.ToolCall.Name) {
		return nil, nil
	}
	flagged := approval.EvaluateFlagged(req)
	if !flagged.Flagged {
		return nil, nil
	}

	reviewer := deps.Reviewer
	if reviewer == nil {
		built, buildErr := NewApprovalReviewer(ctx)
		if buildErr != nil {
			flog.Warn("[chat-agent] approval reviewer unavailable session=%s: %v", deps.SessionID, buildErr)
			metrics.Agent().IncApprovalVerdict("error")
			return escalateAutoApproval(ctx, deps, event, flagged.Reason+" (reviewer unavailable)")
		}
		reviewer = built
	}

	reviewCtx, cancel := context.WithTimeout(ctx, approvalReviewTimeout())
	defer cancel()
	review, reviewErr := reviewer.Review(reviewCtx, approval.ReviewRequest{
		ToolName:      event.ToolCall.Name,
		Args:          event.Args,
		FlaggedReason: flagged.Reason,
	})
	if reviewErr != nil {
		flog.Warn("[chat-agent] approval review failed session=%s tool=%s: %v",
			deps.SessionID, event.ToolCall.Name, reviewErr)
		metrics.Agent().IncApprovalVerdict("error")
		return escalateAutoApproval(ctx, deps, event, flagged.Reason+" (reviewer error)")
	}

	switch review.Verdict {
	case approval.VerdictApprove:
		metrics.Agent().IncApprovalVerdict("approve")
		if deps.Breaker != nil {
			deps.Breaker.Reset()
		}
		return nil, nil
	case approval.VerdictDeny:
		metrics.Agent().IncApprovalVerdict("deny")
		reason := review.Reason
		if reason == "" {
			reason = flagged.Reason
		}
		return denyAutoApproval(deps, reason), nil
	default:
		metrics.Agent().IncApprovalVerdict("escalate")
		return escalateAutoApproval(ctx, deps, event, review.Reason)
	}
}

func evaluateDenyOnlyGate(
	ctx context.Context,
	uid types.Uid,
	event hooks.ToolCallEvent,
) (*hooks.ToolCallResult, permission.Request, error) {
	cfg, err := LoadUserPermissions(ctx, uid)
	if err != nil {
		return &hooks.ToolCallResult{Block: true, Reason: "permission unavailable"}, permission.Request{}, nil
	}
	evaluator := permission.NewEvaluator(cfg)
	workspaceRoot := config.App.ChatAgent.Workspace
	req := permission.Request{
		Tool:          event.ToolCall.Name,
		Args:          event.Args,
		WorkspaceRoot: workspaceRoot,
		ExternalPath:  detectExternalPath(event, workspaceRoot),
	}
	result := evaluator.EvaluateDenyOnly(req)
	if result.Action == permission.ActionDeny {
		return &hooks.ToolCallResult{Block: true, Reason: "permission denied"}, req, nil
	}
	return nil, req, nil
}

// denyAutoApproval blocks after an aux-LLM DENY and advances the denial circuit breaker.
func denyAutoApproval(deps ChatHookDeps, reason string) *hooks.ToolCallResult {
	if reason == "" {
		reason = "approval denied"
	}
	msg := "auto approval denied: " + reason
	if deps.Breaker != nil && deps.Breaker.RecordDenial() {
		msg = approval.ReasonBreakerTripped + " (" + reason + ")"
	}
	return &hooks.ToolCallResult{Block: true, Reason: msg}
}

// blockAutoApproval blocks without advancing the denial circuit breaker (user reject / escalate fallback).
func blockAutoApproval(reason string) *hooks.ToolCallResult {
	if reason == "" {
		reason = "approval denied"
	}
	return &hooks.ToolCallResult{Block: true, Reason: "auto approval denied: " + reason}
}

func escalateAutoApproval(
	ctx context.Context,
	deps ChatHookDeps,
	event hooks.ToolCallEvent,
	reason string,
) (*hooks.ToolCallResult, error) {
	gate := resolveRunConfirm(ctx, deps)
	if gate == nil {
		flog.Debug("[chat-agent] auto escalate blocked without confirm gate session=%s tool=%s",
			deps.SessionID, event.ToolCall.Name)
		return &hooks.ToolCallResult{Block: true, Reason: ReasonConfirmRequiredPlatform}, nil
	}
	eval := permission.Result{
		Action:           permission.ActionAsk,
		PermissionKey:    permission.PermissionKeyForTool(event.ToolCall.Name),
		Pattern:          strings.TrimSpace(fmt.Sprint(event.Args["command"])),
		SuggestAlways:    false,
		SuggestedPattern: "",
	}
	if eval.Pattern == "" || eval.Pattern == "<nil>" {
		eval.Pattern = strings.TrimSpace(fmt.Sprint(event.Args["path"]))
	}
	if reason != "" {
		flog.Debug("[chat-agent] auto escalate session=%s tool=%s reason=%s",
			deps.SessionID, event.ToolCall.Name, reason)
	}
	resp, err := gate.Wait(ctx, event, eval)
	if err != nil {
		return blockAutoApproval(err.Error()), nil
	}
	if !resp.Approved {
		return blockAutoApproval("user denied"), nil
	}
	if resp.Mode == ConfirmModeAlways {
		flog.Warn("[chat-agent] always grant rejected in auto mode session=%s tool=%s",
			deps.SessionID, event.ToolCall.Name)
		// Treat as once if approved; do not persist grants.
	}
	if deps.Breaker != nil {
		deps.Breaker.Reset()
	}
	return nil, nil
}

func evaluatePermissionResult(
	ctx context.Context,
	deps ChatHookDeps,
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
		return handlePermissionAsk(ctx, deps, event, result, sessionState)
	default:
		return &hooks.ToolCallResult{Block: true, Reason: "permission denied"}, nil
	}
}

func handlePermissionAsk(
	ctx context.Context,
	deps ChatHookDeps,
	event hooks.ToolCallEvent,
	result permission.Result,
	sessionState *permission.SessionState,
) (*hooks.ToolCallResult, error) {
	sessionID := deps.SessionID
	gate := resolveRunConfirm(ctx, deps)
	if gate == nil {
		flog.Debug("[chat-agent] ask blocked without confirm gate session=%s tool=%s",
			sessionID, event.ToolCall.Name)
		return &hooks.ToolCallResult{Block: true, Reason: ReasonConfirmRequiredPlatform}, nil
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
