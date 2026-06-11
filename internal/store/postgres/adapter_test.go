package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestClient(t *testing.T) *gen.Client {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	return sqlitetest.OpenClient(t, dbName)
}

func testAdapter(t *testing.T) *adapter {
	t.Helper()
	return &adapter{client: getTestClient(t)}
}

func TestListTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seeds   func(*testing.T, *adapter)
		wantLen int
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(_ *testing.T, _ *adapter) {},
			wantLen: 0,
		},
		{
			name: "with valid tokens returns them",
			seeds: func(t *testing.T, a *adapter) {
				token, err := a.CreateToken(context.Background(), types.Uid("user:alice"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				require.NotEmpty(t, token)
				_, err = a.CreateToken(context.Background(), types.Uid("user:bob"), time.Now().Add(7*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
			},
			wantLen: 2,
		},
		{
			name: "filters expired unused tokens older than 30 days",
			seeds: func(t *testing.T, a *adapter) {
				_, err := a.CreateToken(context.Background(), types.Uid("user:old"), time.Now().Add(-40*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				_, err = a.CreateToken(context.Background(), types.Uid("user:recent"), time.Now().Add(24*time.Hour), []string{"pipeline:read"})
				require.NoError(t, err)
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			tt.seeds(t, a)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			if tt.wantLen > 0 {
				for _, item := range items {
					assert.NotEmpty(t, item.Token)
					assert.Contains(t, item.Token, "fb_")
					assert.NotEmpty(t, item.UID)
				}
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		uid       types.Uid
		expiresAt time.Time
		scopes    []string
		wantErr   bool
	}{
		{
			name:      "creates token successfully",
			uid:       types.Uid("user:test"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    []string{"admin:*"},
			wantErr:   false,
		},
		{
			name:      "creates token with multiple scopes",
			uid:       types.Uid("user:multi"),
			expiresAt: time.Now().Add(7 * 24 * time.Hour),
			scopes:    []string{"hub:apps:read", "pipeline:read"},
			wantErr:   false,
		},
		{
			name:      "creates token with past expiry still succeeds",
			uid:       types.Uid("user:expired"),
			expiresAt: time.Now().Add(-1 * time.Hour),
			scopes:    []string{"hub:apps:read"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			token, err := a.CreateToken(context.Background(), tt.uid, tt.expiresAt, tt.scopes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, len(token), 10)
			assert.Contains(t, token, "fb_")
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, tt.uid, items[0].UID)
			assert.Equal(t, tt.scopes, items[0].Scopes)
		})
	}
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seed    func(*testing.T, *adapter) string
		wantErr bool
		errIs   error
	}{
		{
			name: "revokes existing token",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:revoke"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				return token
			},
			wantErr: false,
		},
		{
			name: "returns ErrNotFound for nonexistent token",
			seed: func(_ *testing.T, _ *adapter) string {
				return "fb_nonexistent_token_12345678"
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
		{
			name: "revoking already revoked token returns ErrNotFound",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:twice"), time.Now().Add(24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				err = a.RevokeToken(context.Background(), token)
				require.NoError(t, err)
				return token
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			flag := tt.seed(t, a)
			err := a.RevokeToken(context.Background(), flag)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.errIs)
				return
			}
			require.NoError(t, err)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Empty(t, items)
		})
	}
}

func TestCreatePlatformUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		item       *gen.PlatformUser
		wantEmail  string
		wantAvatar string
	}{
		{
			name: "preserves provided profile fields",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U123",
				Name:       "alice",
				Email:      "alice@example.com",
				AvatarURL:  "https://example.com/a.png",
				IsBot:      false,
			},
			wantEmail:  "alice@example.com",
			wantAvatar: "https://example.com/a.png",
		},
		{
			name: "fills missing email and avatar placeholders",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U01DMQDTV5W",
				Name:       "user",
				IsBot:      false,
			},
			wantEmail:  "U01DMQDTV5W@unknown.local",
			wantAvatar: "-",
		},
		{
			name: "fills only missing avatar when email is present",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U999",
				Name:       "user",
				Email:      "user@slack.local",
				IsBot:      false,
			},
			wantEmail:  "user@slack.local",
			wantAvatar: "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			id, err := a.CreatePlatformUser(context.Background(), tt.item)
			require.NoError(t, err)
			assert.Positive(t, id)

			created, err := a.client.PlatformUser.Get(context.Background(), id)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, created.Email)
			assert.Equal(t, tt.wantAvatar, created.AvatarURL)
		})
	}
}
