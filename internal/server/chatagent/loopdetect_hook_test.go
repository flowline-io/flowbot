package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/loopdetect"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLoopDetectHookTest(t *testing.T) types.Uid {
	t.Helper()
	LockAppConfigForTest(t)
	installSQLiteTestDatabase(t)
	return types.Uid("user-loop-detect-test")
}

func TestLoopDetectHookTerminate(t *testing.T) {
	uid := setupLoopDetectHookTest(t)

	cfg := loopdetect.DefaultConfig()
	cfg.GlobalCircuitBreaker = 2
	cfg.NoProgressCritical = 100
	cfg.GenericCritical = 100
	cfg.GenericWarn = 100
	detector := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "a.go"}
	res := msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "same"}}}
	detector.ObserveResult(permission.ToolReadFile, args, res) // 1 completed + pending = 2

	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:     "loop-terminate",
		UID:           uid,
		Service:       NewService(),
		DCG:           dcg.AllowAllChecker{},
		LoopDetector:  detector,
		ApprovalMode:  approval.ModeOff,
		WorkspaceRoot: t.TempDir(),
	})

	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     args,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Block)
	assert.True(t, result.Terminate)
	assert.Contains(t, result.Reason, "global circuit breaker")
}

func TestLoopDetectHookWarnDoesNotBlock(t *testing.T) {
	uid := setupLoopDetectHookTest(t)

	cfg := loopdetect.DefaultConfig()
	cfg.GenericWarn = 2
	cfg.GenericCritical = 50
	cfg.NoProgressCritical = 50
	cfg.GlobalCircuitBreaker = 50
	detector := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "a.go"}
	res := msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}}}
	detector.ObserveResult(permission.ToolReadFile, args, res)

	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:     "loop-warn",
		UID:           uid,
		Service:       NewService(),
		DCG:           dcg.AllowAllChecker{},
		LoopDetector:  detector,
		ApprovalMode:  approval.ModeOff,
		WorkspaceRoot: t.TempDir(),
	})

	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     args,
	})
	require.NoError(t, err)
	if result != nil {
		assert.False(t, result.Block, "warn must not block: %v", result.Reason)
	}
}

func TestLoopDetectHookCriticalAskWithoutGate(t *testing.T) {
	uid := setupLoopDetectHookTest(t)

	cfg := loopdetect.DefaultConfig()
	cfg.GenericCritical = 2
	cfg.GenericWarn = 2
	cfg.NoProgressCritical = 50
	cfg.GlobalCircuitBreaker = 50
	detector := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "a.go"}
	res := msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}}}
	detector.ObserveResult(permission.ToolReadFile, args, res)

	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:     "loop-ask",
		UID:           uid,
		Service:       NewService(),
		DCG:           dcg.AllowAllChecker{},
		LoopDetector:  detector,
		ApprovalMode:  approval.ModeManual,
		WorkspaceRoot: t.TempDir(),
	})

	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     args,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Block)
	assert.False(t, result.Terminate)
	assert.Contains(t, result.Reason, "approval")
}

func TestLoopDetectHookPostCompaction(t *testing.T) {
	uid := setupLoopDetectHookTest(t)

	cfg := loopdetect.DefaultConfig()
	cfg.PostCompactionIdentical = 2
	cfg.GenericCritical = 100
	cfg.GenericWarn = 100
	cfg.NoProgressCritical = 100
	cfg.GlobalCircuitBreaker = 100
	detector := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "a.go"}
	res := msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "same"}}}

	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:     "loop-compact",
		UID:           uid,
		Service:       NewService(),
		DCG:           dcg.AllowAllChecker{},
		LoopDetector:  detector,
		ApprovalMode:  approval.ModeOff,
		WorkspaceRoot: t.TempDir(),
	})

	reg.EmitObservation(context.Background(), hooks.ObservationEvent{
		Type: hooks.EventContextCompacted,
	}, func(string, ...any) {})

	detector.ObserveResult(permission.ToolReadFile, args, res)
	detector.ObserveResult(permission.ToolReadFile, args, res)

	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     args,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Block)
	assert.True(t, result.Terminate)
	assert.Contains(t, result.Reason, "compaction")
}
