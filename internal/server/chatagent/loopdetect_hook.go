package chatagent

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/loopdetect"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
)

func registerLoopDetectHooks(reg *hooks.Registry, deps ChatHookDeps) {
	if reg == nil {
		return
	}
	detector := deps.LoopDetector
	if detector == nil || !detector.Config().Enabled {
		return
	}

	hooks.OnToolCall(reg, func(ctx context.Context, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		dec := detector.Check(event.ToolCall.Name, event.Args)
		if !dec.Stuck() {
			return nil, nil
		}
		metrics.Agent().IncLoopDetect(string(dec.Detector), string(dec.Level))
		flog.Info("[chat-agent] loop_detect session=%s tool=%s detector=%s level=%s count=%d terminate=%t",
			deps.SessionID, event.ToolCall.Name, dec.Detector, dec.Level, dec.Count, dec.Terminate)

		if dec.Level == loopdetect.LevelWarn {
			return nil, nil
		}

		if dec.Terminate {
			reason := dec.Message
			if reason == "" {
				reason = "tool loop detected"
			}
			return &hooks.ToolCallResult{Block: true, Reason: reason, Terminate: true}, nil
		}

		return handleLoopDetectCritical(ctx, deps, detector, event, dec)
	})

	hooks.OnToolResult(reg, func(_ context.Context, event hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
		detector.ObserveResult(event.ToolCall.Name, event.Args, event.Result)
		return nil, nil
	})

	hooks.Observe(reg, func(_ context.Context, event hooks.ObservationEvent) error {
		if event.Type == hooks.EventContextCompacted {
			detector.ArmPostCompaction()
			flog.Debug("[chat-agent] loop_detect post-compaction armed session=%s", deps.SessionID)
		}
		return nil
	})
}

func handleLoopDetectCritical(
	ctx context.Context,
	deps ChatHookDeps,
	detector *loopdetect.Detector,
	event hooks.ToolCallEvent,
	dec loopdetect.Decision,
) (*hooks.ToolCallResult, error) {
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
	var sessionState *permission.SessionState
	if deps.Service != nil {
		sessionState = deps.Service.permissionSessions.GetPermissionSession(ctx, deps.SessionID)
	}
	eval := permission.NewEvaluator(cfg)
	result := eval.ResolveDoomLoop(dec.ArgsHash, sessionState)

	switch result.Action {
	case permission.ActionAllow:
		detector.ResetFingerprint(dec.ArgsHash)
		flog.Info("[chat-agent] loop_detect allowed session=%s tool=%s detector=%s",
			deps.SessionID, event.ToolCall.Name, dec.Detector)
		return nil, nil
	case permission.ActionDeny:
		reason := dec.Message
		if reason == "" {
			reason = "permission denied"
		}
		return &hooks.ToolCallResult{Block: true, Reason: reason}, nil
	case permission.ActionAsk:
		out, askErr := evaluatePermissionResult(ctx, deps, event, result, sessionState)
		if askErr != nil {
			return out, askErr
		}
		if out == nil {
			detector.ResetFingerprint(dec.ArgsHash)
			flog.Info("[chat-agent] loop_detect user-allowed session=%s tool=%s detector=%s",
				deps.SessionID, event.ToolCall.Name, dec.Detector)
		}
		return out, nil
	default:
		return &hooks.ToolCallResult{Block: true, Reason: "permission denied"}, nil
	}
}
