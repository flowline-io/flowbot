package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
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
		run  func(*testing.T, *adapter)
	}{
		{
			name: "create get update and soft delete",
			run: func(t *testing.T, a *adapter) {
				usr := &gen.User{
					Flag: "user:alice", Name: "Alice", State: int(schema.UserActive),
					CreatedAt: now, UpdatedAt: now,
				}
				require.NoError(t, a.UserCreate(ctx, usr))
				assert.Positive(t, usr.ID)

				got, err := a.UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, "Alice", got.Name)

				require.NoError(t, a.UserUpdate(ctx, types.Uid("user:alice"), types.KV{"name": "Alice B"}))
				got, err = a.UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, "Alice B", got.Name)

				first, err := a.FirstUser(ctx)
				require.NoError(t, err)
				assert.Equal(t, "user:alice", first.Flag)

				all, err := a.UserGetAll(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				require.Len(t, all, 1)

				require.NoError(t, a.UserDelete(ctx, types.Uid("user:alice"), false))
				got, err = a.UserGet(ctx, types.Uid("user:alice"))
				require.NoError(t, err)
				assert.Equal(t, int(schema.UserInactive), got.State)
			},
		},
		{
			name: "hard delete removes user",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.UserCreate(ctx, &gen.User{
					Flag: "user:bob", Name: "Bob", State: int(schema.UserActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, a.UserDelete(ctx, types.Uid("user:bob"), true))
				_, err := a.UserGet(ctx, types.Uid("user:bob"))
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "get missing user returns not found",
			run: func(t *testing.T, a *adapter) {
				_, err := a.UserGet(ctx, types.Uid("missing"))
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, testAdapter(t))
		})
	}
}

func TestOAuthFormAndParameter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:oauth")
	topic := "github"
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "oauth set get and list available",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.OAuthSet(ctx, gen.OAuth{
					UID: uid.String(), Topic: topic, Type: "github",
					Name: "GitHub", Token: "tok-1", TokenType: "Bearer",
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := a.OAuthGet(ctx, uid, topic, "github")
				require.NoError(t, err)
				assert.Equal(t, "tok-1", got.Token)

				require.NoError(t, a.OAuthSet(ctx, gen.OAuth{
					UID: uid.String(), Topic: topic, Type: "github",
					Name: "GitHub", Token: "tok-2", UpdatedAt: now,
				}))
				got, err = a.OAuthGet(ctx, uid, topic, "github")
				require.NoError(t, err)
				assert.Equal(t, "tok-2", got.Token)

				available, err := a.OAuthGetAvailable(ctx, "github")
				require.NoError(t, err)
				require.Len(t, available, 1)
			},
		},
		{
			name: "form set and get",
			run: func(t *testing.T, a *adapter) {
				formID := "form-1"
				require.NoError(t, a.FormSet(ctx, formID, gen.Form{
					FormID: formID, UID: uid.String(), Topic: topic,
					State:     int(schema.FormStateCreated),
					Schema:    map[string]any{"title": "Settings"},
					Values:    map[string]any{"name": "alice"},
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := a.FormGet(ctx, formID)
				require.NoError(t, err)
				assert.Equal(t, "alice", got.Values["name"])

				require.NoError(t, a.FormSet(ctx, formID, gen.Form{
					FormID: formID, UID: uid.String(), Topic: topic,
					State:  int(schema.FormStateSubmitSuccess),
					Values: map[string]any{"name": "bob"},
				}))
				got, err = a.FormGet(ctx, formID)
				require.NoError(t, err)
				assert.Equal(t, "bob", got.Values["name"])
			},
		},
		{
			name: "parameter set get delete",
			run: func(t *testing.T, a *adapter) {
				flag := "param-1"
				exp := now.Add(time.Hour)
				require.NoError(t, a.ParameterSet(ctx, flag, types.KV{"k": "v"}, exp))
				row, err := a.ParameterGet(ctx, flag)
				require.NoError(t, err)
				assert.Equal(t, "v", types.KV(row.Params)["k"])

				require.NoError(t, a.ParameterDelete(ctx, flag))
				_, err = a.ParameterGet(ctx, flag)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, testAdapter(t))
		})
	}
}

func TestMessageCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "create get by flag platform and session",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateMessage(ctx, gen.Message{
					Flag: "msg-1", PlatformID: 1, PlatformMsgID: "pmsg-1",
					Topic: "general", Role: "user", Session: "sess-1",
					State: int(schema.MessageCreated), Content: map[string]any{"text": "hi"},
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := a.GetMessage(ctx, "msg-1")
				require.NoError(t, err)
				assert.Equal(t, "hi", got.Content["text"])

				byPlatform, err := a.GetMessageByPlatform(ctx, 1, "pmsg-1")
				require.NoError(t, err)
				assert.Equal(t, "msg-1", byPlatform.Flag)

				rows, err := a.GetMessagesBySession(ctx, "sess-1")
				require.NoError(t, err)
				require.Len(t, rows, 1)
			},
		},
		{
			name: "get missing message returns not found",
			run: func(t *testing.T, a *adapter) {
				_, err := a.GetMessage(ctx, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "empty session returns empty slice",
			run: func(t *testing.T, a *adapter) {
				rows, err := a.GetMessagesBySession(ctx, "empty")
				require.NoError(t, err)
				assert.Empty(t, rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, testAdapter(t))
		})
	}
}
