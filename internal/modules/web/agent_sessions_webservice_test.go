package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	start := 0
	if opts.Cursor != "" {
		for i, sess := range s.chatSessions {
			if sess.Flag == opts.Cursor || sess.ID == parseCursorID(opts.Cursor) {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(s.chatSessions) {
		end = len(s.chatSessions)
	}
	page := s.chatSessions[start:end]
	var nextCursor string
	if end < len(s.chatSessions) && len(page) > 0 {
		nextCursor = page[len(page)-1].Flag
	}
	return page, nextCursor, nil
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

func parseCursorID(cursor string) int64 {
	var id int64
	for _, ch := range cursor {
		if ch < '0' || ch > '9' {
			return 0
		}
		id = id*10 + int64(ch-'0')
	}
	return id
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
