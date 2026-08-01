package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedReviewer struct {
	result approval.ReviewResult
	err    error
}

func (f fixedReviewer) Review(context.Context, approval.ReviewRequest) (approval.ReviewResult, error) {
	return f.result, f.err
}

func setupApprovalHookTest(t *testing.T) {
	t.Helper()
	LockAppConfigForTest(t)
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = postgres.NewSQLiteTestAdapter(t)
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		ResetPermissionCacheForTest()
		ResetApprovalCacheForTest()
	})
}

func TestAutoApprovalHook(t *testing.T) {
	setupApprovalHookTest(t)
	uid := types.Uid("user-approval-test")

	tests := []struct {
		name       string
		reviewer   approval.Reviewer
		tool       string
		args       map[string]any
		wantBlock  bool
		wantSubstr string
		confirm    bool
		approve    bool
	}{
		{
			name:      "unflagged write allows",
			reviewer:  fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictDeny, Reason: "should not run"}},
			tool:      permission.ToolWriteFile,
			args:      map[string]any{"path": "pkg/foo.go", "content": "x"},
			wantBlock: false,
		},
		{
			name:       "deny blocks with reason",
			reviewer:   fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictDeny, Reason: "rm risk"}},
			tool:       permission.ToolRunTerminal,
			args:       map[string]any{"command": "rm -rf /tmp/x"},
			wantBlock:  true,
			wantSubstr: "rm risk",
		},
		{
			name:      "approve allows flagged",
			reviewer:  fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictApprove}},
			tool:      permission.ToolRunTerminal,
			args:      map[string]any{"command": "rm -rf /tmp/x"},
			wantBlock: false,
		},
		{
			name:       "reviewer error blocks without gate",
			reviewer:   fixedReviewer{err: assert.AnError},
			tool:       permission.ToolWebFetch,
			args:       map[string]any{"url": "https://example.com"},
			wantBlock:  true,
			wantSubstr: "requires approval",
		},
		{
			name:      "escalate once approve",
			reviewer:  fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictEscalate, Reason: "unsure"}},
			tool:      permission.ToolWebFetch,
			args:      map[string]any{"url": "https://example.com"},
			confirm:   true,
			approve:   true,
			wantBlock: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := hooks.NewRegistry()
			deps := ChatHookDeps{
				SessionID:    "auto-hook-test",
				UID:          uid,
				Service:      NewService(),
				DCG:          dcg.AllowAllChecker{},
				Reviewer:     tt.reviewer,
				ApprovalMode: approval.ModeAuto,
				Breaker:      approval.NewBreaker(3),
			}
			if tt.confirm {
				pub := NewChannelPublisher(8)
				gate := NewConfirmGate(deps.SessionID, pub, nil)
				deps.Confirm = gate
				go func() {
					waitConfirmEvent(t, pub)
					_ = gate.Resolve(ConfirmResponse{
						Approved: tt.approve,
						Reason:   ConfirmReasonApproved,
						Mode:     ConfirmModeAlways,
					})
				}()
			}
			RegisterHooks(reg, deps)
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
				Args:     tt.args,
			})
			require.NoError(t, err)
			if tt.wantBlock {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				if tt.wantSubstr != "" {
					assert.Contains(t, result.Reason, tt.wantSubstr)
				}
				return
			}
			if result != nil {
				assert.False(t, result.Block, "reason=%s", result.Reason)
			}
		})
	}
}

func TestAutoApprovalBreakerLatches(t *testing.T) {
	setupApprovalHookTest(t)
	reg := hooks.NewRegistry()
	breaker := approval.NewBreaker(2)
	deps := ChatHookDeps{
		SessionID:    "breaker-test",
		UID:          types.Uid("u1"),
		Service:      NewService(),
		DCG:          dcg.AllowAllChecker{},
		Reviewer:     fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictDeny, Reason: "no"}},
		ApprovalMode: approval.ModeAuto,
		Breaker:      breaker,
	}
	RegisterHooks(reg, deps)
	event := hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
		Args:     map[string]any{"command": "rm -rf /tmp/x"},
	}
	r1, err := reg.EmitToolCall(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, r1)
	assert.True(t, r1.Block)
	r2, err := reg.EmitToolCall(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, r2)
	assert.Contains(t, r2.Reason, "circuit breaker")
	r3, err := reg.EmitToolCall(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, r3)
	assert.Equal(t, approval.ReasonBreakerTripped, r3.Reason)
}

func TestAutoApprovalBreakerIgnoresEscalateWithoutGate(t *testing.T) {
	setupApprovalHookTest(t)
	reg := hooks.NewRegistry()
	breaker := approval.NewBreaker(2)
	deps := ChatHookDeps{
		SessionID:    "breaker-escalate",
		UID:          types.Uid("u1"),
		Service:      NewService(),
		DCG:          dcg.AllowAllChecker{},
		Reviewer:     fixedReviewer{err: assert.AnError},
		ApprovalMode: approval.ModeAuto,
		Breaker:      breaker,
	}
	RegisterHooks(reg, deps)
	event := hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolWebFetch},
		Args:     map[string]any{"url": "https://example.com"},
	}
	for range 3 {
		r, err := reg.EmitToolCall(context.Background(), event)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.True(t, r.Block)
		assert.NotContains(t, r.Reason, "circuit breaker")
		assert.False(t, breaker.Tripped())
	}
}

func TestOffApprovalDeniesEnv(t *testing.T) {
	setupApprovalHookTest(t)
	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:    "off-test",
		UID:          types.Uid("u1"),
		Service:      NewService(),
		DCG:          dcg.AllowAllChecker{},
		ApprovalMode: approval.ModeOff,
	})
	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     map[string]any{"path": "secrets.env"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Block)
	assert.Equal(t, "permission denied", result.Reason)
}

func TestAutonomousIgnoresAutoMode(t *testing.T) {
	setupApprovalHookTest(t)
	reg := hooks.NewRegistry()
	RegisterHooks(reg, ChatHookDeps{
		SessionID:    "auto-sched",
		UID:          types.Uid("u1"),
		Service:      NewService(),
		Kind:         RunKindScheduled,
		DCG:          dcg.AllowAllChecker{},
		Reviewer:     fixedReviewer{result: approval.ReviewResult{Verdict: approval.VerdictApprove}},
		ApprovalMode: approval.ModeAuto,
	})
	result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
		Args:     map[string]any{"command": "git status"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Block)
}
