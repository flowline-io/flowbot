// Package providers provides provider registry and common provider interfaces.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tidwall/gjson"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// OAuthToken is a typed representation of an OAuth token returned from
// provider token exchange or token refresh flows.
type OAuthToken struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	Scope        string     `json:"scope,omitempty"`
	Extra        any        `json:"extra,omitempty"`
}

// OAuthTokenStore persists OAuth tokens without exposing ORM types.
type OAuthTokenStore interface {
	Get(ctx context.Context, uid types.Uid, topic, provider string) (*OAuthToken, error)
	Set(ctx context.Context, uid types.Uid, topic string, token *OAuthToken) error
}

var (
	oauthTokenStoreMu sync.RWMutex
	oauthTokenStore   OAuthTokenStore
)

// SetOAuthTokenStore wires the persistence backend used by GetOrRefreshToken.
func SetOAuthTokenStore(s OAuthTokenStore) {
	oauthTokenStoreMu.Lock()
	defer oauthTokenStoreMu.Unlock()
	oauthTokenStore = s
}

func getOAuthTokenStore() OAuthTokenStore {
	oauthTokenStoreMu.RLock()
	defer oauthTokenStoreMu.RUnlock()
	return oauthTokenStore
}

// OAuthProvider defines the interface for OAuth authorization providers.
// Providers construct their own authorize URLs and exchange authorization
// codes for typed OAuthToken values.
type OAuthProvider interface {
	GetAuthorizeURL(state string) string
	GetAccessToken(ctx fiber.Ctx) (*OAuthToken, error)
}

// OAuthRefresher is an optional interface implemented by OAuth providers
// whose access tokens expire and can be rotated via a refresh token.
type OAuthRefresher interface {
	RefreshAccessToken(ctx context.Context, refreshToken string) (*OAuthToken, error)
}

// OAuthProviderFactory is a constructor function that returns a new
// OAuthProvider instance, used for provider self-registration.
type OAuthProviderFactory func() OAuthProvider

// oauthRegistry holds the registered OAuth provider factories keyed by
// provider name. Providers register via RegisterOAuthProvider through
// their exported Register() function.
var oauthRegistry = map[string]OAuthProviderFactory{}

// RegisterOAuthProvider registers a provider factory under the given name.
// It is typically called from a provider package's exported Register() function.
func RegisterOAuthProvider(name string, factory OAuthProviderFactory) {
	oauthRegistry[name] = factory
}

// UnregisterOAuthProvider removes an OAuth provider factory from the registry.
func UnregisterOAuthProvider(name string) {
	delete(oauthRegistry, name)
}

// GetOAuthProvider returns a new OAuthProvider instance for the named
// provider. It returns an error if no factory is registered for the name.
func GetOAuthProvider(name string) (OAuthProvider, error) {
	factory, ok := oauthRegistry[name]
	if !ok {
		return nil, fmt.Errorf("providers: unknown oauth provider %q", name)
	}
	return factory(), nil
}

// GetOrRefreshToken retrieves an OAuth token for the given uid, topic, and
// provider type. If the stored token is expired and the provider implements
// OAuthRefresher, it automatically refreshes the token and persists the
// updated values before returning the fresh token.
func GetOrRefreshToken(ctx context.Context, uid types.Uid, topic, t string) (*OAuthToken, error) {
	store := getOAuthTokenStore()
	if store == nil {
		return nil, fmt.Errorf("providers: oauth token store is not configured")
	}

	oauth, err := store.Get(ctx, uid, topic, t)
	if err != nil {
		return nil, err
	}

	if oauth.ExpiresAt != nil && time.Now().After(*oauth.ExpiresAt) {
		provider, err := GetOAuthProvider(t)
		if err != nil {
			return nil, err
		}
		refresher, ok := provider.(OAuthRefresher)
		if !ok {
			return nil, fmt.Errorf("%w: provider %q does not support token refresh", types.ErrForbidden, t)
		}
		newToken, err := refresher.RefreshAccessToken(ctx, oauth.RefreshToken)
		if err != nil {
			return nil, err
		}
		if newToken.Type == "" {
			newToken.Type = t
		}
		if newToken.Name == "" {
			newToken.Name = oauth.Name
		}
		if err := store.Set(ctx, uid, topic, newToken); err != nil {
			flog.Warn("providers: failed to persist refreshed oauth token for %s: %v", t, err)
		}
		return newToken, nil
	}

	return oauth, nil
}

// RedirectURI builds the OAuth callback path for a provider and flag.
func RedirectURI(name, flag string) string {
	return fmt.Sprintf("%s/oauth/%s/%s", types.AppUrl(), name, flag)
}

// Configs holds raw provider configuration JSON.
var Configs json.RawMessage

// ErrMissingConfig is returned when provider configs have not been loaded.
var ErrMissingConfig = fmt.Errorf("provider configs are empty")

// GetConfig reads a nested config value for the named provider.
func GetConfig(name, key string) (gjson.Result, error) {
	if len(Configs) == 0 {
		return gjson.Result{}, ErrMissingConfig
	}
	value := gjson.GetBytes(Configs, fmt.Sprintf("%s.%s", name, key))
	return value, nil
}
