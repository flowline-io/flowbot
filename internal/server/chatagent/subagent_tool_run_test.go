package chatagent

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskToolExecuteWithFakeSubagent(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "subagent output")
	config.App.ChatAgent.SubagentMaxDepth = 2

	ctx := context.Background()
	sessionID := "sess-subagent-tool"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user:alice", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, store.Database.CreateAgentSubagent(ctx, &gen.AgentSubagent{
		Flag: "helper", Name: "helper", Description: "General helper",
		SystemPrompt: "You are a helper.", Source: "test", Enabled: true,
	}))

	ws := coding.Workspace{Root: t.TempDir()}
	tool := NewTaskTool(ws, TaskToolDeps{SessionID: sessionID, UID: "user:alice"})

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
		wantSub string
	}{
		{
			name: "delegates to stored subagent",
			args: map[string]any{
				"subagent_type": "helper",
				"description":   "quick task",
				"prompt":        "summarize logs",
			},
			wantSub: "subagent output",
		},
		{
			name: "unknown subagent returns tool error",
			args: map[string]any{
				"subagent_type": "missing",
				"description":   "noop",
				"prompt":        "noop",
			},
			wantErr: true,
			wantSub: "unknown subagent",
		},
		{
			name: "disabled subagent lookup fails",
			args: map[string]any{
				"subagent_type": "disabled-one",
				"description":   "noop",
				"prompt":        "noop",
			},
			wantErr: true,
			wantSub: "unknown subagent",
		},
	}

	require.NoError(t, store.Database.CreateAgentSubagent(ctx, &gen.AgentSubagent{
		Flag: "disabled-one", Name: "disabled-one", Description: "off",
		SystemPrompt: "off", Source: "test", Enabled: false,
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, "call-1", tt.args, nil)
			require.NoError(t, err)
			text := subagentToolResultText(result)
			if tt.wantErr {
				assert.True(t, result.IsError)
				assert.Contains(t, text, tt.wantSub)
				return
			}
			assert.False(t, result.IsError)
			assert.Contains(t, text, tt.wantSub)

			tasks, listErr := store.Database.ListAgentSubagentTasks(ctx, sessionID, 5)
			require.NoError(t, listErr)
			require.NotEmpty(t, tasks)
			assert.Equal(t, subagentTaskStatusCompleted, tasks[0].Status)
		})
	}
}

func subagentToolResultText(result msg.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = out.WriteString(tp.Text)
		}
	}
	return out.String()
}
