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

func TestScheduledRunPermissionOverlay(t *testing.T) {
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
		tool    string
		args    map[string]any
		wantBlk bool
	}{
		{
			name:    "scheduled run blocks bash",
			kind:    RunKindScheduled,
			tool:    permission.ToolRunTerminal,
			args:    map[string]any{"command": "ls"},
			wantBlk: true,
		},
		{
			name:    "scheduled run allows read file",
			kind:    RunKindScheduled,
			tool:    permission.ToolReadFile,
			args:    map[string]any{"path": "note.txt"},
			wantBlk: false,
		},
		{
			name:    "interactive run blocks bash without gate",
			kind:    RunKindInteractive,
			tool:    permission.ToolRunTerminal,
			args:    map[string]any{"command": "ls"},
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
				ToolCall: msg.ToolCallPart{Name: tt.tool},
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
