package web

import (
	"cmp"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func (s *testStore) ListChatSessions(_ context.Context, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
	if s.chatSessionsErr != nil {
		return nil, "", s.chatSessionsErr
	}
	page, cursor := listWebTestChatSessions(s.chatSessions, opts)
	return page, cursor, nil
}

func listWebTestChatSessions(sessions []*gen.ChatSession, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows := append([]*gen.ChatSession(nil), sessions...)
	slices.SortFunc(rows, func(a, b *gen.ChatSession) int {
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
	return rows, nextCursor
}

func (s *testStore) GetChatSession(_ context.Context, flag string) (*gen.ChatSession, error) {
	if s.chatSessionsByFlag != nil {
		sess, ok := s.chatSessionsByFlag[flag]
		if !ok {
			return nil, types.ErrNotFound
		}
		return sess, nil
	}
	for _, sess := range s.chatSessions {
		if sess.Flag == flag {
			return sess, nil
		}
	}
	return nil, types.ErrNotFound
}

func (s *testStore) ListChatSessionEntries(_ context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	if s.chatSessionEntriesErr != nil {
		return nil, s.chatSessionEntriesErr
	}
	if s.chatSessionEntries == nil {
		return nil, nil
	}
	return s.chatSessionEntries[sessionID], nil
}

func (s *testStore) GetChatSessionEntryInSession(_ context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error) {
	entries, ok := s.chatSessionEntries[sessionID]
	if !ok {
		return nil, types.ErrNotFound
	}
	for _, entry := range entries {
		if entry.Flag == flag {
			return entry, nil
		}
	}
	return nil, types.ErrNotFound
}

func TestChatSessionStateLabel(t *testing.T) {
	tests := []struct {
		name  string
		state int
		want  string
	}{
		{name: "active session", state: int(schema.ChatSessionActive), want: "Active"},
		{name: "closed session", state: int(schema.ChatSessionClosed), want: "Closed"},
		{name: "unknown session", state: 0, want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatSessionStateLabel(tt.state))
		})
	}
}

func TestAgentSessionsPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "page redirects to login", path: "/service/web/agent-sessions"},
		{name: "list redirects to login", path: "/service/web/agent-sessions/list"},
		{name: "detail redirects to login", path: "/service/web/agent-sessions/sess-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		})
	}
}

func TestAgentSessionsListAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name     string
		path     string
		sessions []*gen.ChatSession
		wantBody string
	}{
		{
			name: "list page contains table",
			path: "/service/web/agent-sessions",
			sessions: []*gen.ChatSession{
				{Flag: "sess-demo", UID: "user:a", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: `data-testid="agent-sessions-table"`,
		},
		{
			name: "table partial renders session row",
			path: "/service/web/agent-sessions/list",
			sessions: []*gen.ChatSession{
				{Flag: "sess-table", UID: "user:b", State: int(schema.ChatSessionClosed), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: "sess-table",
		},
		{
			name:     "empty list shows placeholder",
			path:     "/service/web/agent-sessions/list",
			sessions: nil,
			wantBody: "No sessions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{chatSessions: tt.sessions}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}
}

func TestAgentSessionDetailAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		path       string
		sessions   map[string]*gen.ChatSession
		entries    map[string][]*gen.ChatSessionEntry
		wantStatus int
		wantBody   string
	}{
		{
			name: "detail renders session and entries",
			path: "/service/web/agent-sessions/sess-detail",
			sessions: map[string]*gen.ChatSession{
				"sess-detail": {Flag: "sess-detail", UID: "user:x", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			entries: map[string][]*gen.ChatSessionEntry{
				"sess-detail": {
					{Flag: "entry-1", SessionID: "sess-detail", EntryType: "message", CreatedAt: now, Payload: map[string]any{"role": "user"}},
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "user",
		},
		{
			name:       "missing session returns not found",
			path:       "/service/web/agent-sessions/missing",
			sessions:   map[string]*gen.ChatSession{},
			wantStatus: http.StatusNotFound,
			wantBody:   "session not found",
		},
		{
			name: "detail shows empty entries message",
			path: "/service/web/agent-sessions/sess-empty",
			sessions: map[string]*gen.ChatSession{
				"sess-empty": {Flag: "sess-empty", UID: "user:y", State: int(schema.ChatSessionClosed), UpdatedAt: now, CreatedAt: now},
			},
			entries:    map[string][]*gen.ChatSessionEntry{},
			wantStatus: http.StatusOK,
			wantBody:   "No entries in this session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				chatSessionsByFlag: tt.sessions,
				chatSessionEntries: tt.entries,
			}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}
}

func TestAgentSessionEntryPayloadAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		path       string
		entries    map[string][]*gen.ChatSessionEntry
		wantStatus int
		wantBody   string
	}{
		{
			name: "payload partial renders json",
			path: "/service/web/agent-sessions/sess-1/entries/entry-1/payload",
			entries: map[string][]*gen.ChatSessionEntry{
				"sess-1": {
					{Flag: "entry-1", SessionID: "sess-1", EntryType: "message", CreatedAt: now, Payload: map[string]any{"role": "user"}},
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "role",
		},
		{
			name:       "missing entry shows not found",
			path:       "/service/web/agent-sessions/sess-1/entries/missing/payload",
			entries:    map[string][]*gen.ChatSessionEntry{},
			wantStatus: http.StatusOK,
			wantBody:   "Entry not found",
		},
		{
			name: "empty payload shows placeholder",
			path: "/service/web/agent-sessions/sess-2/entries/entry-2/payload",
			entries: map[string][]*gen.ChatSessionEntry{
				"sess-2": {
					{Flag: "entry-2", SessionID: "sess-2", EntryType: "custom", CreatedAt: now},
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "No payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{chatSessionEntries: tt.entries}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}
}

func TestListWebTestChatSessions(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	tests := []struct {
		name       string
		sessions   []*gen.ChatSession
		opts       store.ListChatSessionsOptions
		wantLen    int
		wantCursor bool
		wantFirst  int64
	}{
		{
			name:     "empty slice returns empty page",
			sessions: nil,
			opts:     store.ListChatSessionsOptions{Limit: 10},
			wantLen:  0,
		},
		{
			name: "orders by updated_at desc then id desc",
			sessions: []*gen.ChatSession{
				{ID: 1, Flag: "old", UpdatedAt: now.Add(-time.Hour)},
				{ID: 2, Flag: "new", UpdatedAt: now},
			},
			opts:      store.ListChatSessionsOptions{Limit: 10},
			wantLen:   2,
			wantFirst: 2,
		},
		{
			name: "cursor uses numeric session id",
			sessions: []*gen.ChatSession{
				{ID: 10, Flag: "a", UpdatedAt: now},
				{ID: 20, Flag: "b", UpdatedAt: now.Add(time.Minute)},
				{ID: 30, Flag: "c", UpdatedAt: now.Add(2 * time.Minute)},
			},
			opts:       store.ListChatSessionsOptions{Limit: 2},
			wantLen:    2,
			wantCursor: true,
			wantFirst:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, cursor := listWebTestChatSessions(tt.sessions, tt.opts)
			assert.Len(t, page, tt.wantLen)
			if tt.wantFirst != 0 {
				require.NotEmpty(t, page)
				assert.Equal(t, tt.wantFirst, page[0].ID)
			}
			if tt.wantCursor {
				assert.NotEmpty(t, cursor)
				_, err := strconv.ParseInt(cursor, 10, 64)
				require.NoError(t, err)

				page2, cursor2 := listWebTestChatSessions(tt.sessions, store.ListChatSessionsOptions{
					Limit:  tt.opts.Limit,
					Cursor: cursor,
				})
				require.NotEmpty(t, page2)
				assert.NotEqual(t, page[0].ID, page2[0].ID)
				assert.Empty(t, cursor2)
			}
		})
	}
}
