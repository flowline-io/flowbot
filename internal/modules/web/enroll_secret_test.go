package web

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestEnrollSecretRoundTrip(t *testing.T) {
	enc, _, _, err := webauth.LoadEncryptor("test-encryption-key-for-unit-tests", t.TempDir())
	require.NoError(t, err)
	setWebEncryptor(enc)

	token := "pending-enroll-token"
	params := types.KV{
		"uid": "user-admin", "username": "admin", "topic": "web", "kind": webauth.KindPendingEnroll,
	}
	pending := &pendingSession{
		Kind:     webauth.KindPendingEnroll,
		UID:      "user-admin",
		Username: "admin",
		Token:    token,
	}

	dbClient := sqlitetest.OpenClient(t, t.Name())
	ts := &testStore{dbClient: dbClient}
	require.NoError(t, store.NewModuleDataStore(dbClient).ParameterSet(
		context.Background(),
		auth.HashToken(token),
		params,
		time.Now().Add(time.Minute),
	))

	oldDB := store.Database
	store.Database = ts
	wireNotifyStoresForTest(t)
	defer func() { store.Database = oldDB; setWebEncryptor(nil) }()

	require.NoError(t, stashEnrollSecret(nil, pending, "MYTOTSECRETKEY123456"))
	gotParam, err := store.NewModuleDataStore(dbClient).ParameterGet(context.Background(), auth.HashToken(token))
	require.NoError(t, err)
	stored := types.KV(gotParam.Params)
	got, err := readEnrollSecret(stored)
	require.NoError(t, err)
	require.Equal(t, "MYTOTSECRETKEY123456", got)
	_, hasLegacy := stored["enroll_secret"]
	require.False(t, hasLegacy)
}
