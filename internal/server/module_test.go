package server

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

// testModuleHandler implements module.Handler for testing registerModules.
type testModuleHandler struct {
	module.Base
	ready bool
}

func (h *testModuleHandler) IsReady() bool { return h.ready }
func (*testModuleHandler) Init(_ json.RawMessage) error {
	return nil
}
func (*testModuleHandler) Register() error  { return nil }
func (*testModuleHandler) Bootstrap() error { return nil }
func (*testModuleHandler) Rules() []any     { return nil }
func (*testModuleHandler) Command(_ types.Context, _ any) (types.MsgPayload, error) {
	return nil, nil
}

func listTestChatSessions(sessions map[string]*gen.ChatSession, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows := filterTestChatSessions(sessions, opts)
	slices.SortFunc(rows, func(a, b *gen.ChatSession) int {
		if opts.PinnedFirst {
			if a.Pinned != b.Pinned {
				if a.Pinned {
					return -1
				}
				return 1
			}
		}
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})

	if opts.Cursor != "" {
		cursorID, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil {
			filtered := rows[:0]
			for _, sess := range rows {
				if sess.ID < cursorID {
					filtered = append(filtered, sess)
				}
			}
			rows = filtered
		}
	}

	var nextCursor string
	if len(rows) > limit {
		nextCursor = strconv.FormatInt(rows[limit-1].ID, 10)
		rows = rows[:limit]
	}
	return rows, nextCursor, nil
}

func filterTestChatSessions(sessions map[string]*gen.ChatSession, opts store.ListChatSessionsOptions) []*gen.ChatSession {
	flagSet := map[string]struct{}{}
	for _, flag := range opts.Flags {
		flagSet[flag] = struct{}{}
	}
	rows := make([]*gen.ChatSession, 0, len(sessions))
	for _, sess := range sessions {
		if opts.UID != "" && sess.UID != opts.UID {
			continue
		}
		if opts.State != nil && sess.State != *opts.State {
			continue
		}
		if opts.Archived != nil && sess.Archived != *opts.Archived {
			continue
		}
		if len(flagSet) > 0 {
			if _, ok := flagSet[sess.Flag]; !ok {
				continue
			}
		}
		rows = append(rows, sess)
	}
	return rows
}

func TestRegisterModules_CreatesNewBot(t *testing.T) {
	setupSQLiteTestDB(t)

	module.Register("test-create-mod-bot-001", &testModuleHandler{ready: false})
	t.Cleanup(func() { module.Unregister("test-create-mod-bot-001") })
	registerModules()

	bot, err := store.PlatformStoreFromDB().GetBotByName(context.Background(), "test-create-mod-bot-001")
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, "test-create-mod-bot-001", bot.Name)
	assert.Equal(t, int(schema.BotInactive), bot.State)
}

func TestRegisterModules_DeactivatesStaleBot(t *testing.T) {
	setupSQLiteTestDB(t)
	seedTestBot(t, &gen.Bot{Name: "stale-bot", State: int(schema.BotActive)})

	registerModules()

	bot, err := store.PlatformStoreFromDB().GetBotByName(context.Background(), "stale-bot")
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, int(schema.BotInactive), bot.State)
}

func TestRegisterModules_SetsActiveForReadyModule(t *testing.T) {
	setupSQLiteTestDB(t)

	module.Register("test-ready-mod-bot-002", &testModuleHandler{ready: true})
	t.Cleanup(func() { module.Unregister("test-ready-mod-bot-002") })
	registerModules()

	bot, err := store.PlatformStoreFromDB().GetBotByName(context.Background(), "test-ready-mod-bot-002")
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, int(schema.BotActive), bot.State)
}

func TestRegisterModules_UpdatesExistingBotState(t *testing.T) {
	setupSQLiteTestDB(t)
	seedTestBot(t, &gen.Bot{Name: "existing-ready-bot", State: int(schema.BotInactive)})

	module.Register("existing-ready-bot", &testModuleHandler{ready: true})
	t.Cleanup(func() { module.Unregister("existing-ready-bot") })
	registerModules()

	bot, err := store.PlatformStoreFromDB().GetBotByName(context.Background(), "existing-ready-bot")
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, int(schema.BotActive), bot.State)
}

func TestListTestChatSessions(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		seeds      []*gen.ChatSession
		opts       store.ListChatSessionsOptions
		wantLen    int
		wantCursor bool
		wantFirst  int64
	}{
		{
			name:    "empty store returns empty slice",
			seeds:   nil,
			opts:    store.ListChatSessionsOptions{Limit: 10},
			wantLen: 0,
		},
		{
			name: "returns newest sessions first",
			seeds: []*gen.ChatSession{
				{ID: 1, Flag: "old", UpdatedAt: now.Add(-time.Hour)},
				{ID: 2, Flag: "new", UpdatedAt: now},
			},
			opts:      store.ListChatSessionsOptions{Limit: 10},
			wantLen:   2,
			wantFirst: 2,
		},
		{
			name: "numeric cursor paginates by id",
			seeds: []*gen.ChatSession{
				{ID: 1, Flag: "a", UpdatedAt: now},
				{ID: 2, Flag: "b", UpdatedAt: now.Add(time.Minute)},
				{ID: 3, Flag: "c", UpdatedAt: now.Add(2 * time.Minute)},
			},
			opts:       store.ListChatSessionsOptions{Limit: 2},
			wantLen:    2,
			wantCursor: true,
			wantFirst:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := make(map[string]*gen.ChatSession, len(tt.seeds))
			for _, sess := range tt.seeds {
				sessions[sess.Flag] = sess
			}

			page, cursor, err := listTestChatSessions(sessions, tt.opts)
			require.NoError(t, err)
			assert.Len(t, page, tt.wantLen)
			if tt.wantFirst != 0 {
				require.NotEmpty(t, page)
				assert.Equal(t, tt.wantFirst, page[0].ID)
			}
			if tt.wantCursor {
				assert.NotEmpty(t, cursor)
				_, err := strconv.ParseInt(cursor, 10, 64)
				require.NoError(t, err)

				page2, cursor2, err := listTestChatSessions(sessions, store.ListChatSessionsOptions{
					Limit:  tt.opts.Limit,
					Cursor: cursor,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, page2)
				assert.NotEqual(t, page[0].Flag, page2[0].Flag)
				assert.Empty(t, cursor2)
			}
		})
	}
}
