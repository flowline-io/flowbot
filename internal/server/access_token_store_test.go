package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/auth"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// tokenTestAdapter is a minimal store.Adapter that only exposes an ent client via GetDB/GetClient,
// matching the BDD stub pattern used by web page specs.
type tokenTestAdapter struct {
	client *gen.Client
}

func (*tokenTestAdapter) Open(pkgconfig.StoreType) error { return nil }
func (*tokenTestAdapter) Close() error                   { return nil }
func (*tokenTestAdapter) IsOpen() bool                   { return true }
func (*tokenTestAdapter) GetName() string                { return "token-test" }
func (*tokenTestAdapter) Stats() any                     { return nil }
func (a *tokenTestAdapter) GetDB() any                   { return a.client }
func (a *tokenTestAdapter) GetClient() *gen.Client       { return a.client }
func (*tokenTestAdapter) Ping(context.Context) (time.Duration, error) {
	return 0, nil
}

func TestWireAccessTokenStore_LookupSeesSeededToken(t *testing.T) {
	client := sqlitetest.OpenClient(t, "access_token_wire")
	origDB := store.Database
	t.Cleanup(func() { store.Database = origDB })
	store.Database = &tokenTestAdapter{client: client}

	raw := "bdd-wire-token"
	require.NoError(t, store.NewModuleDataStore(client).ParameterSet(
		context.Background(),
		auth.HashToken(raw),
		types.KV{"uid": "u1", "topic": "t", "scopes": []string{auth.ScopeAdmin}},
		time.Now().Add(time.Hour),
	))

	t.Cleanup(func() { route.SetAccessTokenStore(nil) })

	route.SetAccessTokenStore(nil)
	_, err := route.LookupAccessToken(context.Background(), raw)
	require.ErrorIs(t, err, types.ErrNotFound, "nil AccessTokenStore must fail lookup")

	WireAccessTokenStore()
	got, err := route.LookupAccessToken(context.Background(), raw)
	require.NoError(t, err)
	require.Positive(t, got.ID)
	params := types.KV(got.Params)
	uid, ok := params.String("uid")
	require.True(t, ok)
	require.Equal(t, "u1", uid)
}
