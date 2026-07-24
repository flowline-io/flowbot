package chatagent

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPermissionsView(t *testing.T) {
	svc := NewService()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() {
		store.Database = origDB
		ResetPermissionCacheForTest()
		svc.ResetPermissionSessionsForTest()
	})

	ctx := context.Background()
	uid := types.Uid("user-perm")
	sessionID := "sess-perm-view"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: uid.String(), State: int(schema.ChatSessionActive),
	}))

	state := svc.permissionSessions.GetPermissionSession(ctx, sessionID)
	require.NoError(t, state.AddGrant("bash", "git status*"))
	PersistSessionGrants(ctx, sessionID, state)

	tests := []struct {
		name      string
		sessionID string
		wantGrant bool
	}{
		{name: "without session omits grants", sessionID: "", wantGrant: false},
		{name: "with session includes grants", sessionID: sessionID, wantGrant: true},
		{name: "unknown session has empty grants", sessionID: "missing-session", wantGrant: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := svc.BuildPermissionsView(ctx, uid, tt.sessionID)
			require.NoError(t, err)
			assert.NotEmpty(t, view.Defaults)
			assert.NotEmpty(t, view.Effective)
			if tt.wantGrant {
				assert.Contains(t, view.SessionGrants["bash"], "git status*")
				return
			}
			assert.Empty(t, view.SessionGrants)
		})
	}
}

func TestParsePermissionsBody(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		wantKey string
		wantErr bool
	}{
		{
			name:    "valid override",
			raw:     []byte(`{"bash":{"default":"deny"}}`),
			wantKey: "bash",
		},
		{
			name:    "invalid json",
			raw:     []byte(`{`),
			wantErr: true,
		},
		{
			name:    "invalid default allow on bash",
			raw:     []byte(`{"bash":"allow"}`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParsePermissionsBody(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			_, ok := cfg[tt.wantKey]
			assert.True(t, ok)
		})
	}
}

func TestClearSessionPermissionGrants(t *testing.T) {
	svc := NewService()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() {
		store.Database = origDB
		svc.ResetPermissionSessionsForTest()
	})

	ctx := context.Background()
	sessionID := "sess-clear-grants"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))

	state := svc.permissionSessions.GetPermissionSession(ctx, sessionID)
	require.NoError(t, state.AddGrant("bash", "ls*"))
	PersistSessionGrants(ctx, sessionID, state)

	svc.ClearSessionPermissionGrants(ctx, sessionID)

	svc.ResetPermissionSessionsForTest()
	reloaded := svc.permissionSessions.GetPermissionSession(ctx, sessionID)
	assert.False(t, reloaded.MatchesGrant("bash", "ls"))
}

func TestSaveAndDeleteUserPermissions(t *testing.T) {
	svc := NewService()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() {
		store.Database = origDB
		ResetPermissionCacheForTest()
	})

	ctx := context.Background()
	uid := types.Uid("user-save")

	cfg := permission.Config{"bash": {Default: permission.ActionDeny}}
	require.NoError(t, SaveUserPermissions(ctx, uid, cfg))

	view, err := svc.BuildPermissionsView(ctx, uid, "")
	require.NoError(t, err)
	assert.Equal(t, permission.ActionDeny, view.User["bash"].Default)

	require.NoError(t, DeleteUserPermissions(ctx, uid))
	view, err = svc.BuildPermissionsView(ctx, uid, "")
	require.NoError(t, err)
	assert.Empty(t, view.User)

	data, err := sonic.Marshal(view.Effective)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}
