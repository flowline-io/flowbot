package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestWebAccountStoreCreateFirstAndConflict(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	ws := store.NewWebAccountStore(client)

	hash, err := webauth.HashPassword("flowbot-dev-pass")
	require.NoError(t, err)

	row, err := ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     "admin",
		PasswordHash: hash,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "admin", row.Username)
	assert.Equal(t, "user-admin", row.UID)
	assert.False(t, row.TotpEnabled)

	n, err := ws.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     "other",
		PasswordHash: hash,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrConflict)
}

func TestWebAccountStoreSoleAccount(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T, ws *store.WebAccountStore)
		wantOK  bool
		wantUID string
	}{
		{
			name:   "empty",
			seed:   func(*testing.T, *store.WebAccountStore) {},
			wantOK: false,
		},
		{
			name: "one account",
			seed: func(t *testing.T, ws *store.WebAccountStore) {
				t.Helper()
				createTestWebAccount(t, ws, "admin", true)
			},
			wantOK:  true,
			wantUID: "user-admin",
		},
		{
			name: "two accounts",
			seed: func(t *testing.T, ws *store.WebAccountStore) {
				t.Helper()
				createTestWebAccount(t, ws, "admin", true)
				createTestWebAccount(t, ws, "other", false)
			},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqlitetest.OpenClient(t, t.Name())
			ws := store.NewWebAccountStore(client)
			tt.seed(t, ws)

			row, ok, err := ws.SoleAccount(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, row)
				assert.Equal(t, tt.wantUID, row.UID)
			}
		})
	}
}

func createTestWebAccount(t *testing.T, ws *store.WebAccountStore, username string, first bool) {
	t.Helper()
	ctx := context.Background()
	hash, err := webauth.HashPassword("flowbot-dev-pass")
	require.NoError(t, err)
	if first {
		_, err = ws.CreateFirstAccount(ctx, store.CreateAccountInput{
			Username: username, PasswordHash: hash,
		})
		require.NoError(t, err)
		return
	}
	uid := webauth.UIDForUsername(username)
	_, err = ws.Client().WebAccount.Create().
		SetUsername(username).
		SetUID(uid).
		SetPasswordHash(hash).
		SetTotpEnabled(false).
		SetBackupCodeHashes([]string{}).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureUser(ctx, uid, username))
}
