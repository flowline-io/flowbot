package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// accessTokenStore adapts ModuleDataStore parameters to route.AccessTokenStore.
type accessTokenStore struct{}

func (accessTokenStore) Get(ctx context.Context, flag string) (route.AccessToken, error) {
	if store.Database == nil {
		return route.AccessToken{}, types.ErrNotFound
	}
	p, err := store.ModuleDataStoreFromDB().ParameterGet(ctx, flag)
	if err != nil {
		return route.AccessToken{}, err
	}
	return genParameterToAccessToken(p), nil
}

func (accessTokenStore) Set(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	return store.ModuleDataStoreFromDB().ParameterSet(ctx, flag, params, expiredAt)
}

func (accessTokenStore) SetParams(ctx context.Context, flag string, params types.KV) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	return store.ModuleDataStoreFromDB().ParameterUpdateParams(ctx, flag, params)
}

func (accessTokenStore) Delete(ctx context.Context, flag string) error {
	if store.Database == nil {
		return nil
	}
	return store.ModuleDataStoreFromDB().ParameterDelete(ctx, flag)
}

func genParameterToAccessToken(p gen.Parameter) route.AccessToken {
	return route.AccessToken{
		ID:        p.ID,
		Flag:      p.Flag,
		Params:    p.Params,
		ExpiredAt: p.ExpiredAt,
	}
}

// WireAccessTokenStore injects the store-backed access token adapter into route.
func WireAccessTokenStore() {
	route.SetAccessTokenStore(accessTokenStore{})
}
