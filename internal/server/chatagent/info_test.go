package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAgentInfo(t *testing.T) {
	LockAppConfigForTest(t)

	origDB := store.Database
	origCfg := config.App
	store.Database = postgres.NewSQLiteTestAdapter(t)
	ws := t.TempDir()
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{
			ChatModel: "gpt-test",
			ToolModel: "gpt-tool",
			Workspace: ws,
		},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "test", ModelNames: []string{"gpt-test"}},
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		config.App = origCfg
	})

	ctx := context.Background()
	require.NoError(t, store.Database.CreateAgentSkill(ctx, &gen.AgentSkill{
		Name: "deploy", Description: "Deploy services", Content: "skill body", Enabled: true, Flag: "skill-deploy",
	}))
	require.NoError(t, store.Database.CreateAgentSubagent(ctx, &gen.AgentSubagent{
		Name: "researcher", Description: "Research helper", SystemPrompt: "You research.",
		Enabled: true, Flag: "sub-research",
	}))

	tests := []struct {
		name      string
		wantSkill int
		checkMeta bool
	}{
		{name: "loads tools skills and subagents", wantSkill: 1},
		{name: "includes version and tool metadata", checkMeta: true},
		{name: "reports provider from config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := BuildAgentInfo(ctx)
			require.NoError(t, err)
			assert.Equal(t, "gpt-test", info.ChatModel)
			assert.Equal(t, "openai", info.Provider)
			assert.Equal(t, ws, info.Workspace)
			assert.NotEmpty(t, info.Tools)
			if tt.wantSkill > 0 {
				assert.GreaterOrEqual(t, info.SkillCount, tt.wantSkill)
				assert.GreaterOrEqual(t, info.SubagentCount, 1)
			}
			if tt.checkMeta {
				assert.NotEmpty(t, info.Version)
				assert.Equal(t, len(info.Tools), info.ToolCount)
			}
		})
	}
}

func TestResolveModelProvider(t *testing.T) {
	LockAppConfigForTest(t)
	orig := config.App.Models
	config.App.Models = []config.Model{
		{Provider: "anthropic", ModelNames: []string{"claude-3"}},
		{Provider: "openai", ModelNames: []string{"gpt-4"}},
	}
	t.Cleanup(func() { config.App.Models = orig })

	tests := []struct {
		name  string
		model string
		want  string
	}{
		{name: "known model", model: "claude-3", want: "anthropic"},
		{name: "other provider", model: "gpt-4", want: "openai"},
		{name: "unknown model", model: "missing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveModelProvider(tt.model))
		})
	}
}

func TestListUserActiveSessions(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	uid := types.Uid("user-sessions")
	now := time.Now().UTC()
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-active", UID: uid.String(), Title: "Active chat",
		State: int(schema.ChatSessionActive), CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-closed", UID: uid.String(), State: int(schema.ChatSessionClosed), CreatedAt: now, UpdatedAt: now,
	}))

	tests := []struct {
		name    string
		uid     types.Uid
		limit   int
		wantLen int
		wantErr bool
	}{
		{name: "lists active sessions for owner", uid: uid, limit: 10, wantLen: 1},
		{name: "other user empty", uid: types.Uid("other"), limit: 10, wantLen: 0},
		{name: "nil database errors", uid: uid, limit: 10, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				store.Database = nil
				t.Cleanup(func() { store.Database = postgres.NewSQLiteTestAdapter(t) })
			}
			rows, _, err := ListUserActiveSessions(ctx, tt.uid, tt.limit, "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, rows, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "sess-active", rows[0].SessionID)
				assert.Equal(t, ModeNormal, rows[0].Mode)
				assert.Equal(t, "active", rows[0].State)
			}
		})
	}
}
