package store

import (
	"context"
	"testing"
	"time"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "create get update and soft delete",
			run: func(t *testing.T, client *gen.Client) {
				usr := &gen.User{
					Flag: "user:alice", Name: "Alice", State: int(schema.UserActive),
					CreatedAt: now, UpdatedAt: now,
				}
				require.NoError(t, NewUserStore(client).UserCreate(ctx, usr))
				assert.Positive(t, usr.ID)

				got, err := NewUserStore(client).UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, "Alice", got.Name)

				require.NoError(t, NewUserStore(client).UserUpdate(ctx, types.Uid("user:alice"), types.KV{"name": "Alice B"}))
				got, err = NewUserStore(client).UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, "Alice B", got.Name)

				first, err := NewUserStore(client).FirstUser(ctx)
				require.NoError(t, err)
				assert.Equal(t, "user:alice", first.Flag)

				all, err := NewUserStore(client).UserGetAll(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				require.Len(t, all, 1)

				require.NoError(t, NewUserStore(client).UserDelete(ctx, types.Uid("user:alice"), false))
				got, err = NewUserStore(client).UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, int(schema.UserInactive), got.State)
			},
		},
		{
			name: "hard delete removes user",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewUserStore(client).UserCreate(ctx, &gen.User{
					Flag: "user:bob", Name: "Bob", State: int(schema.UserActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewUserStore(client).UserDelete(ctx, types.Uid("user:bob"), true))
				_, err := NewUserStore(client).UserGet(ctx, types.Uid("user:bob"))
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "get missing user returns not found",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewUserStore(client).UserGet(ctx, types.Uid("missing"))
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, sqlitetest.OpenClient(t, t.Name()))
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
			client := sqlitetest.OpenClient(t, t.Name())
			id, err := NewUserStore(client).CreatePlatformUser(context.Background(), tt.item)
			require.NoError(t, err)
			assert.Positive(t, id)

			created, err := client.PlatformUser.Get(context.Background(), id)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, created.Email)
			assert.Equal(t, tt.wantAvatar, created.AvatarURL)
		})
	}
}
