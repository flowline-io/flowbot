package web

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
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

	ts := &testStore{}
	ts.paramGetFn = func(_ context.Context, flag string) (gen.Parameter, error) {
		if flag == auth.HashToken(token) {
			return gen.Parameter{ID: 1, Flag: flag, Params: params, ExpiredAt: time.Now().Add(time.Minute)}, nil
		}
		return gen.Parameter{}, types.ErrNotFound
	}
	var stored types.KV
	ts.paramSetFn = func(_ context.Context, _ string, p types.KV, _ time.Time) error {
		stored = p
		params = p
		return nil
	}
	oldDB := store.Database
	store.Database = ts
	defer func() { store.Database = oldDB; setWebEncryptor(nil) }()

	require.NoError(t, stashEnrollSecret(nil, pending, "MYTOTSECRETKEY123456"))
	got, err := readEnrollSecret(stored)
	require.NoError(t, err)
	require.Equal(t, "MYTOTSECRETKEY123456", got)
	_, hasLegacy := stored["enroll_secret"]
	require.False(t, hasLegacy)
}
