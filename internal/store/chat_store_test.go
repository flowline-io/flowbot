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

func TestChatSessionEntryCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "create list get and append updates leaf",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-entry", UID: "user:e", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, NewChatStore(client).CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "e1", SessionID: "sess-entry", EntryType: "message",
					Payload: map[string]any{"role": "user"},
				}))
				require.NoError(t, NewChatStore(client).AppendChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "e2", SessionID: "sess-entry", ParentID: "e1", EntryType: "message",
					Payload: map[string]any{"role": "assistant"},
				}))

				rows, err := NewChatStore(client).ListChatSessionEntries(ctx, "sess-entry")
				require.NoError(t, err)
				require.Len(t, rows, 2)

				got, err := NewChatStore(client).GetChatSessionEntry(ctx, "e2")
				require.NoError(t, err)
				assert.Equal(t, "assistant", got.Payload["role"])

				inSession, err := NewChatStore(client).GetChatSessionEntryInSession(ctx, "sess-entry", "e1")
				require.NoError(t, err)
				assert.Equal(t, "e1", inSession.Flag)

				sess, err := NewChatStore(client).GetChatSession(ctx, "sess-entry")
				require.NoError(t, err)
				assert.Equal(t, "e2", sess.LeafID)
			},
		},
		{
			name: "list entries by sessions returns matching rows",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-a", UID: "user:batch", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-b", UID: "user:batch", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, NewChatStore(client).CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "a1", SessionID: "sess-a", EntryType: "message",
					Payload: map[string]any{"role": "user"},
				}))
				require.NoError(t, NewChatStore(client).CreateChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "b1", SessionID: "sess-b", EntryType: "message",
					Payload: map[string]any{"role": "assistant"},
				}))

				rows, err := NewChatStore(client).ListChatSessionEntriesBySessions(ctx, []string{"sess-a", "sess-b"})
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
			run: func(t *testing.T, client *gen.Client) {
				rows, err := NewChatStore(client).ListChatSessionEntriesBySessions(ctx, nil)
				require.NoError(t, err)
				assert.Nil(t, rows)
			},
		},
		{
			name: "list entries by sessions unknown session",
			run: func(t *testing.T, client *gen.Client) {
				rows, err := NewChatStore(client).ListChatSessionEntriesBySessions(ctx, []string{"missing-sess"})
				require.NoError(t, err)
				assert.Empty(t, rows)
			},
		},
		{
			name: "update mode leaf title and close session",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
					Flag: "sess-upd", UID: "user:u", State: int(schema.ChatSessionActive),
				}))
				require.NoError(t, NewChatStore(client).UpdateChatSessionMode(ctx, "sess-upd", "plan"))
				require.NoError(t, NewChatStore(client).UpdateChatSessionLeaf(ctx, "sess-upd", "leaf-1"))
				require.NoError(t, NewChatStore(client).UpdateChatSessionTitle(ctx, "sess-upd", "Deploy"))
				require.NoError(t, NewChatStore(client).CloseChatSession(ctx, "sess-upd"))

				sess, err := NewChatStore(client).GetChatSession(ctx, "sess-upd")
				require.NoError(t, err)
				assert.Equal(t, "plan", sess.Mode)
				assert.Equal(t, "leaf-1", sess.LeafID)
				assert.Equal(t, "Deploy", sess.Title)
				assert.Equal(t, int(schema.ChatSessionClosed), sess.State)
			},
		},
		{
			name: "append to missing session returns not found",
			run: func(t *testing.T, client *gen.Client) {
				err := NewChatStore(client).AppendChatSessionEntry(ctx, &gen.ChatSessionEntry{
					Flag: "orphan", SessionID: "missing", EntryType: "message",
				})
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

func TestListChatSessions(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name       string
		seeds      func(*testing.T, *gen.Client)
		opts       ListChatSessionsOptions
		wantLen    int
		wantCursor bool
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(_ *testing.T, _ *gen.Client) {},
			opts:    ListChatSessionsOptions{Limit: 10},
			wantLen: 0,
		},
		{
			name: "returns seeded sessions newest first",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-old", UID: "user:a", State: int(schema.ChatSessionActive),
					CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-new", UID: "user:b", State: int(schema.ChatSessionClosed),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts:    ListChatSessionsOptions{Limit: 10},
			wantLen: 2,
		},
		{
			name: "cursor paginates remaining sessions",
			seeds: func(t *testing.T, client *gen.Client) {
				for i := range 3 {
					flag := "sess-" + string(rune('a'+i))
					require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
						Flag: flag, UID: "user:p", State: int(schema.ChatSessionActive),
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
						UpdatedAt: now.Add(time.Duration(i) * time.Minute),
					}))
				}
			},
			opts:       ListChatSessionsOptions{Limit: 2},
			wantLen:    2,
			wantCursor: true,
		},
		{
			name: "uid filter returns only matching owner",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-a", UID: "user:alice", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-b", UID: "user:bob", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts:    ListChatSessionsOptions{Limit: 10, UID: "user:alice"},
			wantLen: 1,
		},
		{
			name: "state filter returns only active sessions",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-active", UID: "user:s", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-closed", UID: "user:s", State: int(schema.ChatSessionClosed),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts: func() ListChatSessionsOptions {
				active := int(schema.ChatSessionActive)
				return ListChatSessionsOptions{Limit: 10, State: &active}
			}(),
			wantLen: 1,
		},
		{
			name: "archived filter excludes archived by default path",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-open", UID: "user:arch", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-arch", UID: "user:arch", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).UpdateChatSessionArchived(context.Background(), "sess-arch", true))
			},
			opts: func() ListChatSessionsOptions {
				archived := false
				return ListChatSessionsOptions{Limit: 10, UID: "user:arch", Archived: &archived}
			}(),
			wantLen: 1,
		},
		{
			name: "pinned first sorts pinned ahead",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-unpin", UID: "user:pin", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-pin", UID: "user:pin", State: int(schema.ChatSessionActive),
					CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
				}))
				require.NoError(t, NewChatStore(client).UpdateChatSessionPinned(context.Background(), "sess-pin", true))
			},
			opts:    ListChatSessionsOptions{Limit: 10, UID: "user:pin", PinnedFirst: true},
			wantLen: 2,
		},
		{
			name: "flags filter returns matching sessions only",
			seeds: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-keep", UID: "user:f", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-drop", UID: "user:f", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts:    ListChatSessionsOptions{Limit: 10, Flags: []string{"sess-keep"}},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			tt.seeds(t, client)

			got, cursor, err := NewChatStore(client).ListChatSessions(context.Background(), tt.opts)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			if tt.name == "pinned first sorts pinned ahead" && len(got) == 2 {
				assert.Equal(t, "sess-pin", got[0].Flag)
				assert.True(t, got[0].Pinned)
			}
			if tt.name == "flags filter returns matching sessions only" && len(got) == 1 {
				assert.Equal(t, "sess-keep", got[0].Flag)
			}
			if tt.wantCursor {
				assert.NotEmpty(t, cursor)
				page2, cursor2, err := NewChatStore(client).ListChatSessions(context.Background(), ListChatSessionsOptions{
					Limit:  tt.opts.Limit,
					Cursor: cursor,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, page2)
				assert.Empty(t, cursor2)
				return
			}
			assert.Empty(t, cursor)
		})
	}
}

func TestCountChatSessions(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	client := sqlitetest.OpenClient(t, t.Name())
	require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
		Flag: "count-active", UID: "user:count", State: int(schema.ChatSessionActive),
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
		Flag: "count-closed", UID: "user:count", State: int(schema.ChatSessionClosed),
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, NewChatStore(client).CreateChatSession(context.Background(), &gen.ChatSession{
		Flag: "count-other", UID: "user:other", State: int(schema.ChatSessionActive),
		CreatedAt: now, UpdatedAt: now,
	}))

	tests := []struct {
		name string
		opts ListChatSessionsOptions
		want int
	}{
		{name: "all sessions", opts: ListChatSessionsOptions{}, want: 3},
		{
			name: "active only",
			opts: func() ListChatSessionsOptions {
				active := int(schema.ChatSessionActive)
				return ListChatSessionsOptions{State: &active}
			}(),
			want: 2,
		},
		{name: "by uid", opts: ListChatSessionsOptions{UID: "user:count"}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewChatStore(client).CountChatSessions(context.Background(), tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateChatSessionTitle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := sqlitetest.OpenClient(t, t.Name())

	require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-title", UID: "user:t", State: int(schema.ChatSessionActive),
	}))

	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{name: "sets title", title: "Deploy flowbot"},
		{name: "updates title", title: "Redis configuration"},
		{name: "missing session", title: "ghost", wantErr: types.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := "sess-title"
			if tt.wantErr != nil {
				flag = "missing-session"
			}
			err := NewChatStore(client).UpdateChatSessionTitle(ctx, flag, tt.title)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			got, err := NewChatStore(client).GetChatSession(ctx, "sess-title")
			require.NoError(t, err)
			assert.Equal(t, tt.title, got.Title)
		})
	}
}

func TestUpdateChatSessionListMeta(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := sqlitetest.OpenClient(t, t.Name())
	require.NoError(t, NewChatStore(client).CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-meta", UID: "user:m", State: int(schema.ChatSessionActive),
	}))

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "sets preview",
			run: func(t *testing.T) {
				require.NoError(t, NewChatStore(client).UpdateChatSessionPreview(ctx, "sess-meta", "last reply"))
				got, err := NewChatStore(client).GetChatSession(ctx, "sess-meta")
				require.NoError(t, err)
				assert.Equal(t, "last reply", got.Preview)
			},
		},
		{
			name: "pins and unpins",
			run: func(t *testing.T) {
				require.NoError(t, NewChatStore(client).UpdateChatSessionPinned(ctx, "sess-meta", true))
				got, err := NewChatStore(client).GetChatSession(ctx, "sess-meta")
				require.NoError(t, err)
				assert.True(t, got.Pinned)
				require.NoError(t, NewChatStore(client).UpdateChatSessionPinned(ctx, "sess-meta", false))
				got, err = NewChatStore(client).GetChatSession(ctx, "sess-meta")
				require.NoError(t, err)
				assert.False(t, got.Pinned)
			},
		},
		{
			name: "archives and restores",
			run: func(t *testing.T) {
				require.NoError(t, NewChatStore(client).UpdateChatSessionArchived(ctx, "sess-meta", true))
				got, err := NewChatStore(client).GetChatSession(ctx, "sess-meta")
				require.NoError(t, err)
				assert.True(t, got.Archived)
				require.NoError(t, NewChatStore(client).UpdateChatSessionArchived(ctx, "sess-meta", false))
				got, err = NewChatStore(client).GetChatSession(ctx, "sess-meta")
				require.NoError(t, err)
				assert.False(t, got.Archived)
			},
		},
		{
			name: "missing session preview",
			run: func(t *testing.T) {
				err := NewChatStore(client).UpdateChatSessionPreview(ctx, "missing", "x")
				assert.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestChatScheduledTaskStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	runAt := now.Add(2 * time.Hour)

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "create list and update task",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-1",
					UID:          "user:alice",
					Name:         "daily",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 9 * * *",
					Prompt:       "check logs",
					State:        string(schema.ChatScheduledTaskStateActive),
					CreatedAt:    now,
					UpdatedAt:    now,
				}))
				rows, err := NewChatStore(client).ListChatScheduledTasks(ctx, ListChatScheduledTasksOptions{UID: "user:alice"})
				require.NoError(t, err)
				require.Len(t, rows, 1)

				prompt := "updated prompt"
				require.NoError(t, NewChatStore(client).UpdateChatScheduledTask(ctx, "task-1", UpdateChatScheduledTaskParams{Prompt: &prompt}))
				got, err := NewChatStore(client).GetChatScheduledTaskForUID(ctx, "task-1", "user:alice")
				require.NoError(t, err)
				assert.Equal(t, prompt, got.Prompt)
			},
		},
		{
			name: "create once task run record",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-once",
					UID:          "user:bob",
					Name:         "reminder",
					ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
					Prompt:       "submit report",
					RunAt:        &runAt,
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, NewChatStore(client).CreateChatScheduledTaskRun(ctx, &gen.ChatScheduledTaskRun{
					Flag:         "run-1",
					TaskID:       "task-once",
					RunSessionID: "sess-run",
					State:        string(schema.ChatScheduledTaskRunStateCompleted),
					Reply:        "done",
					StartedAt:    now,
				}))
				runs, err := NewChatStore(client).ListChatScheduledTaskRuns(ctx, "task-once", 10)
				require.NoError(t, err)
				require.Len(t, runs, 1)
				assert.Equal(t, "done", runs[0].Reply)
			},
		},
		{
			name: "delete task",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-delete",
					UID:          "user:alice",
					Name:         "temp",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 7 * * *",
					Prompt:       "temp",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, NewChatStore(client).DeleteChatScheduledTask(ctx, "task-delete"))
				_, err := NewChatStore(client).GetChatScheduledTask(ctx, "task-delete")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "fail stale running task runs",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-stale",
					UID:          "user:alice",
					Name:         "stale",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 6 * * *",
					Prompt:       "stale",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, NewChatStore(client).CreateChatScheduledTaskRun(ctx, &gen.ChatScheduledTaskRun{
					Flag:         "run-stale",
					TaskID:       "task-stale",
					RunSessionID: "sess-stale",
					State:        string(schema.ChatScheduledTaskRunStateRunning),
					StartedAt:    now,
				}))
				require.NoError(t, NewChatStore(client).FailStaleChatScheduledTaskRuns(ctx))
				runs, err := NewChatStore(client).ListChatScheduledTaskRuns(ctx, "task-stale", 5)
				require.NoError(t, err)
				require.Len(t, runs, 1)
				assert.Equal(t, string(schema.ChatScheduledTaskRunStateFailed), runs[0].State)
				assert.NotEmpty(t, runs[0].Error)
			},
		},
		{
			name: "uid scoped get returns not found for other user",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewChatStore(client).CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-private",
					UID:          "user:owner",
					Name:         "private",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 8 * * *",
					Prompt:       "secret",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				_, err := NewChatStore(client).GetChatScheduledTaskForUID(ctx, "task-private", "user:other")
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
