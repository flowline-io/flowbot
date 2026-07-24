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
		wantBlk bool
	}{
		{name: "memory get allowed", tool: memoryGetToolName, wantBlk: false},
		{name: "memory list allowed", tool: memoryListToolName, wantBlk: false},
		{name: "search summaries allowed", tool: searchSessionSummariesToolName, wantBlk: false},
		{name: "memory set blocked", tool: memorySetToolName, wantBlk: true},
		{name: "memory delete blocked", tool: memoryDeleteToolName, wantBlk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
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
	svc := NewService()
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
		svc.ResetPermissionSessionsForTest()
	})

	tests := []struct {
		name    string
		kind    RunKind
		wantBlk bool
	}{
		{name: "pipeline allows memory set", kind: RunKindPipeline, wantBlk: false},
		{name: "scheduled allows memory set", kind: RunKindScheduled, wantBlk: false},
		{name: "interactive blocks memory set without gate", kind: RunKindInteractive, wantBlk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
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
				Service:   svc,
			})
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: permission.ToolMemorySet},
				Args:     map[string]any{"key": "k", "value": "v"},
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
