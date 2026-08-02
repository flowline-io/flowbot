package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

// oauthTokenStore adapts ModuleDataStore OAuth APIs to providers.OAuthTokenStore.
type oauthTokenStore struct{}

func (oauthTokenStore) Get(ctx context.Context, uid types.Uid, topic, provider string) (*providers.OAuthToken, error) {
	row, err := store.ModuleDataStoreFromDB().OAuthGet(ctx, uid, topic, provider)
	if err != nil {
		return nil, err
	}
	return genOAuthToToken(row), nil
}

func (oauthTokenStore) Set(ctx context.Context, uid types.Uid, topic string, token *providers.OAuthToken) error {
	if token == nil {
		return fmt.Errorf("oauth token is nil")
	}
	existing, err := store.ModuleDataStoreFromDB().OAuthGet(ctx, uid, topic, token.Type)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}

	row := gen.OAuth{
		UID:   uid.String(),
		Topic: topic,
		Type:  token.Type,
		Name:  token.Name,
		Token: token.AccessToken,
	}
	if err == nil {
		row = existing
		if token.Name != "" {
			row.Name = token.Name
		}
		if token.Type != "" {
			row.Type = token.Type
		}
		if token.AccessToken != "" {
			row.Token = token.AccessToken
		}
	}
	if token.RefreshToken != "" {
		row.RefreshToken = token.RefreshToken
	}
	if token.ExpiresAt != nil {
		row.ExpiresAt = *token.ExpiresAt
	}
	if token.TokenType != "" {
		row.TokenType = token.TokenType
	}
	if token.Scope != "" {
		row.Scope = token.Scope
	}
	if token.Extra != nil {
		if m, ok := token.Extra.(map[string]any); ok {
			row.Extra = m
		} else {
			row.Extra = map[string]any{"extra": token.Extra}
		}
	}
	now := time.Now()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	return store.ModuleDataStoreFromDB().OAuthSet(ctx, row)
}

func genOAuthToToken(o gen.OAuth) *providers.OAuthToken {
	var expiresAt *time.Time
	if !o.ExpiresAt.IsZero() {
		t := o.ExpiresAt
		expiresAt = &t
	}
	return &providers.OAuthToken{
		Name:         o.Name,
		Type:         o.Type,
		AccessToken:  o.Token,
		RefreshToken: o.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    o.TokenType,
		Scope:        o.Scope,
		Extra:        o.Extra,
	}
}

// WireOAuthTokenStore injects the store-backed OAuth token adapter into providers.
func WireOAuthTokenStore() {
	providers.SetOAuthTokenStore(oauthTokenStore{})
}
