package server

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatAgentPermissionHookAskWithoutGateBlocks(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		chatagent.ResetPermissionCacheForTest()
		ChatAgentService().ResetPermissionSessionsForTest()
	})

	tests := []struct {
		name       string
		tool       string
		args       map[string]any
		wantBlk    bool
		wantReason string
	}{
		{
			name:       "bash ask without gate blocks",
			tool:       permission.ToolRunTerminal,
			args:       map[string]any{"command": "ls"},
			wantBlk:    true,
			wantReason: chatagent.ReasonConfirmRequiredPlatform,
		},
		{name: "read env denied", tool: permission.ToolReadFile, args: map[string]any{"path": "secrets.env"}, wantBlk: true},
		{name: "skill allow", tool: permission.ToolReadSkill, args: map[string]any{"name": "demo"}, wantBlk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := hooks.NewRegistry()
			chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{
				SessionID: "sess-1",
				UID:       types.Uid("user-1"),
				DCG:       dcg.AllowAllChecker{},
				Service:   ChatAgentService(),
			})
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
				Args:     tt.args,
			})
			require.NoError(t, err)
			if tt.wantBlk {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				if tt.wantReason != "" {
					assert.Contains(t, result.Reason, tt.wantReason)
				}
				return
			}
			if result != nil {
				assert.False(t, result.Block)
			}
		})
	}
}

func TestChatAgentPermissionHookAlwaysGrantUsesSuggestedPattern(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		chatagent.ResetPermissionCacheForTest()
		ChatAgentService().ResetPermissionSessionsForTest()
		ChatAgentService().ClearAPIRunState("sess-1", nil)
	})
	chatagent.ResetPermissionCacheForTest()
	ChatAgentService().ResetPermissionSessionsForTest()

	pub := chatagent.NewChannelPublisher(8)
	gate := chatagent.NewConfirmGate("sess-1", pub, nil)
	state := chatagent.NewAPIRunState(pub, gate)
	require.NoError(t, ChatAgentService().TrySetAPIRunState("sess-1", state))
	t.Cleanup(func() { ChatAgentService().ClearAPIRunState("sess-1", state) })

	reg := hooks.NewRegistry()
	chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{
		SessionID: "sess-1",
		UID:       types.Uid("user-1"),
		DCG:       dcg.AllowAllChecker{},
		Service:   ChatAgentService(),
		Publisher: pub,
		Confirm:   gate,
	})

	done := make(chan *hooks.ToolCallResult, 1)
	go func() {
		result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
			Args:     map[string]any{"command": "git status"},
		})
		assert.NoError(t, err)
		done <- result
	}()

	waitConfirmEvent := func(t *testing.T) {
		t.Helper()
		select {
		case ev := <-pub.Events():
			assert.Equal(t, chatagent.EventTypeConfirm, ev.Type)
		case <-time.After(time.Second):
			t.Fatal("expected confirm event")
		}
	}
	waitConfirmEvent(t)

	_, err := ChatAgentService().ResolveConfirm("sess-1", gate.ID(), true, chatagent.ConfirmModeAlways, "git *", chatagent.ConfirmReasonApproved)
	require.NoError(t, err)

	result := <-done
	if result != nil {
		assert.False(t, result.Block)
	}

	view, err := ChatAgentService().BuildPermissionsView(context.Background(), types.Uid("user-1"), "sess-1")
	require.NoError(t, err)
	assert.Empty(t, view.SessionGrants["bash"])
}
