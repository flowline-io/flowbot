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
