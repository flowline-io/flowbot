package store

import (
	"context"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestListTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seeds   func(*testing.T, *gen.Client)
		wantLen int
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(_ *testing.T, _ *gen.Client) {},
			wantLen: 0,
		},
		{
			name: "with valid tokens returns them",
			seeds: func(t *testing.T, client *gen.Client) {
				token, err := NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:alice"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				require.NotEmpty(t, token)
				_, err = NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:bob"), time.Now().Add(7*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
			},
			wantLen: 2,
		},
		{
			name: "filters expired unused tokens older than 30 days",
			seeds: func(t *testing.T, client *gen.Client) {
				_, err := NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:old"), time.Now().Add(-40*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				_, err = NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:recent"), time.Now().Add(24*time.Hour), []string{"pipeline:read"})
				require.NoError(t, err)
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			tt.seeds(t, client)
			items, err := NewModuleDataStore(client).ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			if tt.wantLen > 0 {
				for _, item := range items {
					assert.NotEmpty(t, item.Token)
					assert.Len(t, item.Token, 64)
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
		{
			name:      "rejects empty scopes",
			uid:       types.Uid("user:noscope"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    nil,
			wantErr:   true,
		},
		{
			name:      "rejects empty scope slice",
			uid:       types.Uid("user:empty"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    []string{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			token, err := NewModuleDataStore(client).CreateToken(context.Background(), tt.uid, tt.expiresAt, tt.scopes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, len(token), 10)
			assert.Contains(t, token, "fb_")
			items, err := NewModuleDataStore(client).ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, auth.HashToken(token), items[0].Token)
			assert.Equal(t, tt.uid, items[0].UID)
			assert.Equal(t, tt.scopes, items[0].Scopes)
		})
	}
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seed    func(*testing.T, *gen.Client) string
		wantErr bool
		errIs   error
	}{
		{
			name: "revokes existing token",
			seed: func(t *testing.T, client *gen.Client) string {
				token, err := NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:revoke"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				return auth.HashToken(token)
			},
			wantErr: false,
		},
		{
			name: "returns ErrNotFound for nonexistent token",
			seed: func(_ *testing.T, _ *gen.Client) string {
				return "fb_nonexistent_token_12345678"
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
		{
			name: "revoking already revoked token returns ErrNotFound",
			seed: func(t *testing.T, client *gen.Client) string {
				token, err := NewModuleDataStore(client).CreateToken(context.Background(), types.Uid("user:twice"), time.Now().Add(24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				flag := auth.HashToken(token)
				err = NewModuleDataStore(client).RevokeToken(context.Background(), flag)
				require.NoError(t, err)
				return flag
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			flag := tt.seed(t, client)
			err := NewModuleDataStore(client).RevokeToken(context.Background(), flag)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.errIs)
				return
			}
			require.NoError(t, err)
			items, err := NewModuleDataStore(client).ListTokens(context.Background())
			require.NoError(t, err)
			assert.Empty(t, items)
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
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "oauth set get and list available",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewModuleDataStore(client).OAuthSet(ctx, gen.OAuth{
					UID: uid.String(), Topic: topic, Type: "github",
					Name: "GitHub", Token: "tok-1", TokenType: "Bearer",
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := NewModuleDataStore(client).OAuthGet(ctx, uid, topic, "github")
				require.NoError(t, err)
				assert.Equal(t, "tok-1", got.Token)

				require.NoError(t, NewModuleDataStore(client).OAuthSet(ctx, gen.OAuth{
					UID: uid.String(), Topic: topic, Type: "github",
					Name: "GitHub", Token: "tok-2", UpdatedAt: now,
				}))
				got, err = NewModuleDataStore(client).OAuthGet(ctx, uid, topic, "github")
				require.NoError(t, err)
				assert.Equal(t, "tok-2", got.Token)

				available, err := NewModuleDataStore(client).OAuthGetAvailable(ctx, "github")
				require.NoError(t, err)
				require.Len(t, available, 1)
			},
		},
		{
			name: "form set and get",
			run: func(t *testing.T, client *gen.Client) {
				formID := "form-1"
				require.NoError(t, NewModuleDataStore(client).FormSet(ctx, formID, gen.Form{
					FormID: formID, UID: uid.String(), Topic: topic,
					State:     int(schema.FormStateCreated),
					Schema:    map[string]any{"title": "Settings"},
					Values:    map[string]any{"name": "alice"},
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := NewModuleDataStore(client).FormGet(ctx, formID)
				require.NoError(t, err)
				assert.Equal(t, "alice", got.Values["name"])

				require.NoError(t, NewModuleDataStore(client).FormSet(ctx, formID, gen.Form{
					FormID: formID, UID: uid.String(), Topic: topic,
					State:  int(schema.FormStateSubmitSuccess),
					Values: map[string]any{"name": "bob"},
				}))
				got, err = NewModuleDataStore(client).FormGet(ctx, formID)
				require.NoError(t, err)
				assert.Equal(t, "bob", got.Values["name"])
			},
		},
		{
			name: "parameter set get delete",
			run: func(t *testing.T, client *gen.Client) {
				flag := "param-1"
				exp := now.Add(time.Hour)
				require.NoError(t, NewModuleDataStore(client).ParameterSet(ctx, flag, types.KV{"k": "v"}, exp))
				row, err := NewModuleDataStore(client).ParameterGet(ctx, flag)
				require.NoError(t, err)
				assert.Equal(t, "v", types.KV(row.Params)["k"])

				require.NoError(t, NewModuleDataStore(client).ParameterDelete(ctx, flag))
				_, err = NewModuleDataStore(client).ParameterGet(ctx, flag)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "parameter update params keeps expired_at",
			run: func(t *testing.T, client *gen.Client) {
				flag := "param-touch"
				exp := now.Add(3 * time.Hour)
				require.NoError(t, NewModuleDataStore(client).ParameterSet(ctx, flag, types.KV{"k": "v"}, exp))
				require.NoError(t, NewModuleDataStore(client).ParameterUpdateParams(ctx, flag, types.KV{"k": "w"}))
				row, err := NewModuleDataStore(client).ParameterGet(ctx, flag)
				require.NoError(t, err)
				assert.Equal(t, "w", types.KV(row.Params)["k"])
				assert.Equal(t, exp.Unix(), row.ExpiredAt.Unix())
				require.ErrorIs(t, NewModuleDataStore(client).ParameterUpdateParams(ctx, "missing", types.KV{"k": "x"}), types.ErrNotFound)
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

func TestBehaviorCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:behavior")

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "set get list and increase",
			run: func(t *testing.T, client *gen.Client) {
				now := time.Now().UTC().Truncate(time.Second)
				require.NoError(t, NewModuleDataStore(client).BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "msg_in", Count: 2,
					Extra:     map[string]any{"source": "test"},
					CreatedAt: now, UpdatedAt: now,
				}))

				got, err := NewModuleDataStore(client).BehaviorGet(ctx, uid, "msg_in")
				require.NoError(t, err)
				assert.Equal(t, int32(2), got.Count)
				assert.Equal(t, "test", got.Extra["source"])

				require.NoError(t, NewModuleDataStore(client).BehaviorIncrease(ctx, uid, "msg_in", 3))
				got, err = NewModuleDataStore(client).BehaviorGet(ctx, uid, "msg_in")
				require.NoError(t, err)
				assert.Equal(t, int32(5), got.Count)

				rows, err := NewModuleDataStore(client).BehaviorList(ctx, uid)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "msg_in", rows[0].Flag)
			},
		},
		{
			name: "update existing behavior via set",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewModuleDataStore(client).BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "dup", Count: 1,
				}))
				require.NoError(t, NewModuleDataStore(client).BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "dup", Count: 9,
				}))
				got, err := NewModuleDataStore(client).BehaviorGet(ctx, uid, "dup")
				require.NoError(t, err)
				assert.Equal(t, int32(9), got.Count)
			},
		},
		{
			name: "get missing returns not found",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewModuleDataStore(client).BehaviorGet(ctx, uid, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "increase missing row is silent no-op",
			run: func(t *testing.T, client *gen.Client) {
				err := NewModuleDataStore(client).BehaviorIncrease(ctx, uid, "missing", 1)
				require.NoError(t, err)
				rows, err := NewModuleDataStore(client).BehaviorList(ctx, uid)
				require.NoError(t, err)
				assert.Empty(t, rows)
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

func TestConfigAndDataKV(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:cfg")
	topic := "homelab"

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "config set get delete and list",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewModuleDataStore(client).ConfigSet(ctx, uid, topic, "theme", types.KV{"mode": "dark"}))
				got, err := NewModuleDataStore(client).ConfigGet(ctx, uid, topic, "theme")
				require.NoError(t, err)
				assert.Equal(t, "dark", got["mode"])

				require.NoError(t, NewModuleDataStore(client).ConfigSet(ctx, uid, topic, "theme", types.KV{"mode": "light"}))
				got, err = NewModuleDataStore(client).ConfigGet(ctx, uid, topic, "theme")
				require.NoError(t, err)
				assert.Equal(t, "light", got["mode"])

				rows, err := NewModuleDataStore(client).ListConfigByPrefix(ctx, uid, topic, "theme")
				require.NoError(t, err)
				require.Len(t, rows, 1)

				items, err := NewModuleDataStore(client).ListConfigs(ctx, ListConfigOptions{Search: "theme", Limit: 10})
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, "theme", items[0].Key)

				require.NoError(t, NewModuleDataStore(client).ConfigDelete(ctx, uid, topic, "theme"))
				_, err = NewModuleDataStore(client).ConfigGet(ctx, uid, topic, "theme")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "data set get list delete",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewModuleDataStore(client).DataSet(ctx, uid, topic, "note/a", types.KV{"text": "alpha"}))
				require.NoError(t, NewModuleDataStore(client).DataSet(ctx, uid, topic, "note/b", types.KV{"text": "beta"}))

				got, err := NewModuleDataStore(client).DataGet(ctx, uid, topic, "note/a")
				require.NoError(t, err)
				assert.Equal(t, "alpha", got["text"])

				prefix := "note/"
				rows, err := NewModuleDataStore(client).DataList(ctx, uid, topic, types.DataFilter{Prefix: &prefix})
				require.NoError(t, err)
				require.Len(t, rows, 2)

				require.NoError(t, NewModuleDataStore(client).DataDelete(ctx, uid, topic, "note/a"))
				_, err = NewModuleDataStore(client).DataGet(ctx, uid, topic, "note/a")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "config get missing returns not found",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewModuleDataStore(client).ConfigGet(ctx, uid, topic, "missing")
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
