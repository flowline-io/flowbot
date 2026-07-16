package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPlanSummaries(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	sessionID := "sess-plans"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))
	now := time.Now().UTC()
	require.NoError(t, store.Database.CreateAgentPlan(ctx, &gen.AgentPlan{
		Flag: "plan-1", SessionID: sessionID, Title: "Deploy Redis", Content: "steps",
		CreatedAt: now, UpdatedAt: now,
	}))

	tests := []struct {
		name    string
		session string
		wantLen int
		wantErr bool
	}{
		{name: "lists session plans", session: sessionID, wantLen: 1},
		{name: "empty session", session: "missing", wantLen: 0},
		{name: "nil database", session: sessionID, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				store.Database = nil
				t.Cleanup(func() { store.Database = postgres.NewSQLiteTestAdapter(t) })
			}
			rows, err := ListPlanSummaries(ctx, tt.session)
			if tt.wantErr {
				require.ErrorIs(t, err, types.ErrUnavailable)
				return
			}
			require.NoError(t, err)
			assert.Len(t, rows, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "plan-1", rows[0].PlanID)
				assert.Equal(t, "Deploy Redis", rows[0].Title)
			}
		})
	}
}

func TestMaybePersistPlan(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	sessionID := "sess-plan-mode"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, SetSessionMode(ctx, sessionID, ModePlan))

	tests := []struct {
		name   string
		reply  string
		wantOK bool
	}{
		{name: "persists plan in plan mode", reply: "# Rollout\nstep", wantOK: true},
		{name: "skips empty reply", reply: "   ", wantOK: false},
		{name: "skips normal mode", reply: "# Nope", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "skips normal mode" {
				require.NoError(t, SetSessionMode(ctx, sessionID, ModeNormal))
				t.Cleanup(func() { _ = SetSessionMode(ctx, sessionID, ModePlan) })
			}
			id, title, ok := maybePersistPlan(ctx, sessionID, tt.reply)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.NotEmpty(t, id)
				assert.NotEmpty(t, title)
			}
		})
	}
}
