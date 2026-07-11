package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanModeMemoryWriteBlock(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	sessionID := types.Id()
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag:  sessionID,
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, SetSessionMode(ctx, sessionID, ModePlan))

	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		wantBlk bool
	}{
		{name: "memory read allowed", tool: updateMemoryToolName, args: map[string]any{"operation": "read"}, wantBlk: false},
		{name: "memory list allowed", tool: updateMemoryToolName, args: map[string]any{"operation": "list"}, wantBlk: false},
		{name: "memory write blocked", tool: updateMemoryToolName, args: map[string]any{"operation": "write"}, wantBlk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
				Args:     tt.args,
			}
			block := planModeToolBlock(ctx, sessionID, event)
			if tt.wantBlk {
				require.NotNil(t, block)
				assert.True(t, block.Block)
				return
			}
			assert.Nil(t, block)
		})
	}
}

func TestMemoryPermissionOverlay(t *testing.T) {
	LockAppConfigForTest(t)

	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = postgres.NewSQLiteTestAdapter(t)
	root := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: root}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		ResetPermissionCacheForTest()
		ResetPermissionSessionsForTest()
	})

	tests := []struct {
		name    string
		kind    RunKind
		args    map[string]any
		wantBlk bool
	}{
		{
			name:    "pipeline allows memory write",
			kind:    RunKindPipeline,
			args:    map[string]any{"operation": "write", "content": "x"},
			wantBlk: false,
		},
		{
			name:    "scheduled allows memory write",
			kind:    RunKindScheduled,
			args:    map[string]any{"operation": "write", "content": "x"},
			wantBlk: false,
		},
		{
			name:    "interactive blocks memory write without gate",
			kind:    RunKindInteractive,
			args:    map[string]any{"operation": "write", "content": "x"},
			wantBlk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := types.Id()
			require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
				Flag:  sessionID,
				UID:   "user-1",
				State: int(schema.ChatSessionActive),
			}))

			reg := hooks.NewRegistry()
			RegisterHooks(reg, ChatHookDeps{
				SessionID: sessionID,
				UID:       types.Uid("user-1"),
				Kind:      tt.kind,
			})
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: permission.ToolUpdateMemory},
				Args:     tt.args,
			})
			require.NoError(t, err)
			if tt.wantBlk {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				return
			}
			if result != nil {
				assert.False(t, result.Block)
			}
		})
	}
}
