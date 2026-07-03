package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveConfigUsesDefaults(t *testing.T) {
	tests := []struct {
		name string
		user permission.Config
		key  string
	}{
		{name: "empty user keeps bash ask", user: permission.Config{}, key: "bash"},
		{name: "user override", user: permission.Config{"bash": {Default: permission.ActionDeny}}, key: "bash"},
		{name: "read env deny", user: permission.Config{}, key: "read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective := permission.EffectiveConfig(tt.user)
			rs, ok := effective[tt.key]
			assert.True(t, ok)
			if tt.user[tt.key].Default != "" {
				assert.Equal(t, tt.user[tt.key].Default, rs.Default)
			}
		})
	}
}

func TestPermissionSessionManagerPersists(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{name: "same session returns same state", sessionID: "sess-a"},
		{name: "different sessions isolated", sessionID: "sess-b"},
		{name: "empty id still works", sessionID: ""},
	}
	mgr := &PermissionSessionManager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := mgr.GetPermissionSession(context.Background(), tt.sessionID)
			b := mgr.GetPermissionSession(context.Background(), tt.sessionID)
			assert.Equal(t, a, b)
		})
	}
}

func TestSessionGrantsPersistAndClear(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	sessionID := "sess-grants"
	require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
		Flag:  sessionID,
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
	}))
	t.Cleanup(func() {
		store.Database = origDB
		ResetPermissionSessionsForTest()
	})

	ctx := context.Background()
	state := permissionSessions.GetPermissionSession(ctx, sessionID)
	require.NoError(t, state.AddGrant("bash", "git status*"))
	PersistSessionGrants(ctx, sessionID, state)

	ResetPermissionSessionsForTest()
	reloaded := permissionSessions.GetPermissionSession(ctx, sessionID)
	assert.True(t, reloaded.MatchesGrant("bash", "git status"))

	require.NoError(t, CloseSession(ctx, sessionID))
	ResetPermissionSessionsForTest()
	afterClose := permissionSessions.GetPermissionSession(ctx, sessionID)
	assert.False(t, afterClose.MatchesGrant("bash", "git status"))
}
