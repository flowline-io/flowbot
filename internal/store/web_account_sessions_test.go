package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestWebAccountStoreRevokeLegacyWebSessions(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	ctx := context.Background()

	full, err := client.Parameter.Create().
		SetFlag("full-session").
		SetParams(map[string]any{
			"uid": "user-admin", "topic": "web", "kind": webauth.KindFull, "scopes": []string{"admin:*"},
		}).
		SetExpiredAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	legacy, err := client.Parameter.Create().
		SetFlag("legacy-session").
		SetParams(map[string]any{
			"uid": "user-admin", "topic": "web", "scopes": []string{"admin:*"},
		}).
		SetExpiredAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	pending, err := client.Parameter.Create().
		SetFlag("pending-session").
		SetParams(map[string]any{
			"uid": "user-admin", "topic": "web", "kind": webauth.KindPending2FA,
		}).
		SetExpiredAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	ws := store.NewWebAccountStore(client)
	deleted, err := ws.RevokeLegacyWebSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	_, err = client.Parameter.Get(ctx, full.ID)
	require.NoError(t, err)
	_, err = client.Parameter.Get(ctx, legacy.ID)
	assert.True(t, gen.IsNotFound(err))
	_, err = client.Parameter.Get(ctx, pending.ID)
	assert.True(t, gen.IsNotFound(err))

	other, err := client.Parameter.Create().
		SetFlag("other-topic").
		SetParams(map[string]any{"uid": "user-admin", "topic": "other"}).
		SetExpiredAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)
	deleted, err = ws.RevokeLegacyWebSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)
	_, err = client.Parameter.Query().Where(parameter.IDEQ(other.ID)).Only(ctx)
	require.NoError(t, err)
}
