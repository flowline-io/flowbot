package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBehaviorCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:behavior")

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "set get list and increase",
			run: func(t *testing.T, a *adapter) {
				now := time.Now().UTC().Truncate(time.Second)
				require.NoError(t, a.BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "msg_in", Count: 2,
					Extra:     map[string]any{"source": "test"},
					CreatedAt: now, UpdatedAt: now,
				}))

				got, err := a.BehaviorGet(ctx, uid, "msg_in")
				require.NoError(t, err)
				assert.Equal(t, int32(2), got.Count)
				assert.Equal(t, "test", got.Extra["source"])

				require.NoError(t, a.BehaviorIncrease(ctx, uid, "msg_in", 3))
				got, err = a.BehaviorGet(ctx, uid, "msg_in")
				require.NoError(t, err)
				assert.Equal(t, int32(5), got.Count)

				rows, err := a.BehaviorList(ctx, uid)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "msg_in", rows[0].Flag)
			},
		},
		{
			name: "update existing behavior via set",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "dup", Count: 1,
				}))
				require.NoError(t, a.BehaviorSet(ctx, gen.Behavior{
					UID: uid.String(), Flag: "dup", Count: 9,
				}))
				got, err := a.BehaviorGet(ctx, uid, "dup")
				require.NoError(t, err)
				assert.Equal(t, int32(9), got.Count)
			},
		},
		{
			name: "get missing returns not found",
			run: func(t *testing.T, a *adapter) {
				_, err := a.BehaviorGet(ctx, uid, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "increase missing row is silent no-op",
			run: func(t *testing.T, a *adapter) {
				err := a.BehaviorIncrease(ctx, uid, "missing", 1)
				require.NoError(t, err)
				rows, err := a.BehaviorList(ctx, uid)
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

func TestConfigAndDataKV(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:cfg")
	topic := "homelab"

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "config set get delete and list",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.ConfigSet(ctx, uid, topic, "theme", types.KV{"mode": "dark"}))
				got, err := a.ConfigGet(ctx, uid, topic, "theme")
				require.NoError(t, err)
				assert.Equal(t, "dark", got["mode"])

				require.NoError(t, a.ConfigSet(ctx, uid, topic, "theme", types.KV{"mode": "light"}))
				got, err = a.ConfigGet(ctx, uid, topic, "theme")
				require.NoError(t, err)
				assert.Equal(t, "light", got["mode"])

				rows, err := a.ListConfigByPrefix(ctx, uid, topic, "theme")
				require.NoError(t, err)
				require.Len(t, rows, 1)

				items, err := a.ListConfigs(ctx, store.ListConfigOptions{Search: "theme", Limit: 10})
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, "theme", items[0].Key)

				require.NoError(t, a.ConfigDelete(ctx, uid, topic, "theme"))
				_, err = a.ConfigGet(ctx, uid, topic, "theme")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "data set get list delete",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.DataSet(ctx, uid, topic, "note/a", types.KV{"text": "alpha"}))
				require.NoError(t, a.DataSet(ctx, uid, topic, "note/b", types.KV{"text": "beta"}))

				got, err := a.DataGet(ctx, uid, topic, "note/a")
				require.NoError(t, err)
				assert.Equal(t, "alpha", got["text"])

				prefix := "note/"
				rows, err := a.DataList(ctx, uid, topic, types.DataFilter{Prefix: &prefix})
				require.NoError(t, err)
				require.Len(t, rows, 2)

				require.NoError(t, a.DataDelete(ctx, uid, topic, "note/a"))
				_, err = a.DataGet(ctx, uid, topic, "note/a")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "config get missing returns not found",
			run: func(t *testing.T, a *adapter) {
				_, err := a.ConfigGet(ctx, uid, topic, "missing")
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

func TestAgentHostCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:agent-host")
	topic := "default"
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "create get list and update online fields",
			run: func(t *testing.T, a *adapter) {
				id, err := a.CreateAgent(ctx, &gen.Agent{
					UID: uid.String(), Topic: topic, Hostid: "host-1", Hostname: "node-a",
					LastOnlineAt: now, CreatedAt: now, UpdatedAt: now,
				})
				require.NoError(t, err)
				assert.Positive(t, id)

				got, err := a.GetAgentByHostid(ctx, uid, topic, "host-1")
				require.NoError(t, err)
				assert.Equal(t, "node-a", got.Hostname)

				all, err := a.GetAgents(ctx)
				require.NoError(t, err)
				require.Len(t, all, 1)

				onlineAt := now.Add(5 * time.Minute)
				require.NoError(t, a.UpdateAgentLastOnlineAt(ctx, uid, topic, "host-1", onlineAt))

				offlineAt := onlineAt.Add(2 * time.Minute)
				require.NoError(t, a.UpdateAgentOnlineDuration(ctx, uid, topic, "host-1", offlineAt))

				updated, err := a.GetAgentByHostid(ctx, uid, topic, "host-1")
				require.NoError(t, err)
				assert.Equal(t, onlineAt.Unix(), updated.LastOnlineAt.Unix())
				assert.Equal(t, int32(120), updated.OnlineDuration)
			},
		},
		{
			name: "get by hostid missing returns not found",
			run: func(t *testing.T, a *adapter) {
				_, err := a.GetAgentByHostid(ctx, uid, topic, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "update online duration missing returns not found",
			run: func(t *testing.T, a *adapter) {
				err := a.UpdateAgentOnlineDuration(ctx, uid, topic, "ghost", time.Now())
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

func TestChatSessionEntryCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "create list get and append updates leaf",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-entry", UID: "user:e", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, a.CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "e1", SessionID: "sess-entry", EntryType: "message",
					Payload: map[string]any{"role": "user"},
				}))
				require.NoError(t, a.AppendChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "e2", SessionID: "sess-entry", ParentID: "e1", EntryType: "message",
					Payload: map[string]any{"role": "assistant"},
				}))

				rows, err := a.ListChatSessionEntries(ctx, "sess-entry")
				require.NoError(t, err)
				require.Len(t, rows, 2)

				got, err := a.GetChatSessionEntry(ctx, "e2")
				require.NoError(t, err)
				assert.Equal(t, "assistant", got.Payload["role"])

				inSession, err := a.GetChatSessionEntryInSession(ctx, "sess-entry", "e1")
				require.NoError(t, err)
				assert.Equal(t, "e1", inSession.Flag)

				sess, err := a.GetChatSession(ctx, "sess-entry")
				require.NoError(t, err)
				assert.Equal(t, "e2", sess.LeafID)
			},
		},
		{
			name: "list entries by sessions returns matching rows",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-a", UID: "user:batch", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, a.CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-b", UID: "user:batch", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, a.CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "a1", SessionID: "sess-a", EntryType: "message",
					Payload: map[string]any{"role": "user"},
				}))
				require.NoError(t, a.CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "b1", SessionID: "sess-b", EntryType: "message",
					Payload: map[string]any{"role": "assistant"},
				}))

				rows, err := a.ListChatSessionEntriesBySessions(ctx, []string{"sess-a", "sess-b"})
				require.NoError(t, err)
				require.Len(t, rows, 2)
				ids := map[string]string{}
				for _, row := range rows {
					ids[row.Flag] = row.SessionID
				}
				assert.Equal(t, "sess-a", ids["a1"])
				assert.Equal(t, "sess-b", ids["b1"])
			},
		},
		{
			name: "list entries by sessions empty ids",
			run: func(t *testing.T, a *adapter) {
				rows, err := a.ListChatSessionEntriesBySessions(ctx, nil)
				require.NoError(t, err)
				assert.Nil(t, rows)
			},
		},
		{
			name: "list entries by sessions unknown session",
			run: func(t *testing.T, a *adapter) {
				rows, err := a.ListChatSessionEntriesBySessions(ctx, []string{"missing-sess"})
				require.NoError(t, err)
				assert.Empty(t, rows)
			},
		},
		{
			name: "update mode leaf title and close session",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-upd", UID: "user:u", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, a.UpdateChatSessionMode(ctx, "sess-upd", "plan"))
				require.NoError(t, a.UpdateChatSessionLeaf(ctx, "sess-upd", "leaf-1"))
				require.NoError(t, a.UpdateChatSessionTitle(ctx, "sess-upd", "Deploy"))
				require.NoError(t, a.CloseChatSession(ctx, "sess-upd"))

				sess, err := a.GetChatSession(ctx, "sess-upd")
				require.NoError(t, err)
				assert.Equal(t, "plan", sess.Mode)
				assert.Equal(t, "leaf-1", sess.LeafID)
				assert.Equal(t, "Deploy", sess.Title)
				assert.Equal(t, int(schema.ChatSessionClosed), sess.State)
			},
		},
		{
			name: "append to missing session returns not found",
			run: func(t *testing.T, a *adapter) {
				err := a.AppendChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "orphan", SessionID: "missing", EntryType: "message",
				})
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
