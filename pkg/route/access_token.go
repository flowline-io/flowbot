package route

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// AccessToken is the route-layer view of a persisted access-token parameter.
type AccessToken struct {
	ID        int64
	Flag      string
	Params    map[string]any
	ExpiredAt time.Time
}

// AccessTokenIsExpired reports whether the token expiry is before now.
// Matches store.ParameterIsExpired semantics (zero time is treated as expired).
func AccessTokenIsExpired(p AccessToken) bool {
	return p.ExpiredAt.Before(time.Now())
}

// AccessTokenStore persists access-token parameters without exposing ORM types.
type AccessTokenStore interface {
	Get(ctx context.Context, flag string) (AccessToken, error)
	Set(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	// SetParams updates params without changing expired_at.
	SetParams(ctx context.Context, flag string, params types.KV) error
	Delete(ctx context.Context, flag string) error
}

var (
	accessTokenStoreMu sync.RWMutex
	accessTokenStore   AccessTokenStore
)

// SetAccessTokenStore wires the persistence backend used by token lookup helpers.
func SetAccessTokenStore(s AccessTokenStore) {
	accessTokenStoreMu.Lock()
	defer accessTokenStoreMu.Unlock()
	accessTokenStore = s
}

func getAccessTokenStore() AccessTokenStore {
	accessTokenStoreMu.RLock()
	defer accessTokenStoreMu.RUnlock()
	return accessTokenStore
}
