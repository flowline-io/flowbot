package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
)

func withChatAgentEnabled(t *testing.T, fn func()) {
	t.Helper()
	orig := pkgconfig.App.ChatAgent.ChatModel
	pkgconfig.App.ChatAgent.ChatModel = "test-model"
	t.Cleanup(func() {
		pkgconfig.App.ChatAgent.ChatModel = orig
	})
	fn()
}

func TestAgentsPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "page redirects to login", path: "/service/web/agents"},
		{name: "list redirects to login", path: "/service/web/agents/list"},
		{name: "detail redirects to login", path: "/service/web/agents/sess-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		})
	}
}

func TestAgentsPageAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name     string
		path     string
		sessions []*gen.ChatSession
		wantBody string
	}{
		{
			name: "page contains composer and your sessions",
			path: "/service/web/agents",
			sessions: []*gen.ChatSession{
				{Flag: "sess-mine", Title: "My task", UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
				{Flag: "sess-other", Title: "Other", UID: "other", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: "Your sessions",
		},
		{
			name: "list partial filters by uid",
			path: "/service/web/agents/list",
			sessions: []*gen.ChatSession{
				{Flag: "sess-mine", Title: "Visible", UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
				{Flag: "sess-other", Title: "Hidden", UID: "other", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: "Visible",
		},
		{
			name:     "empty list placeholder",
			path:     "/service/web/agents/list",
			sessions: nil,
			wantBody: "No sessions yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{chatSessions: tt.sessions}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
				req.Header.Set("Cookie", "accessToken=test-token")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				text := string(body)
				assert.Contains(t, text, tt.wantBody)
				if tt.name == "list partial filters by uid" {
					assert.NotContains(t, text, "Hidden")
				}
			})
		})
	}
}

func TestAgentsListShowsTotalDuration(t *testing.T) {
	now := time.Now().UTC()
	withChatAgentEnabled(t, func() {
		e1Payload := mustChatSessionEntryPayload(t, session.TreeEntry{
			ID:       "e1",
			Type:     session.EntryMessage,
			ParentID: "",
			Message: msg.AssistantMessage{
				Parts:         []msg.ContentPart{msg.TextPart{Text: "first"}},
				RunDurationMs: 1200,
			},
		})
		e2Payload := mustChatSessionEntryPayload(t, session.TreeEntry{
			ID:       "e2",
			Type:     session.EntryMessage,
			ParentID: "e1",
			Message: msg.AssistantMessage{
				Parts:         []msg.ContentPart{msg.TextPart{Text: "second"}},
				RunDurationMs: 3400,
			},
		})
		ts := &testStore{
			chatSessions: []*gen.ChatSession{{
				Flag:      "sess-dur",
				Title:     "Timed task",
				UID:       "testuser",
				LeafID:    "e2",
				State:     int(schema.ChatSessionActive),
				UpdatedAt: now,
				CreatedAt: now,
			}},
			chatSessionEntries: map[string][]*gen.ChatSessionEntry{
				"sess-dur": {
					{Flag: "e1", SessionID: "sess-dur", ParentID: "", EntryType: "message", Payload: e1Payload},
					{Flag: "e2", SessionID: "sess-dur", ParentID: "e1", EntryType: "message", Payload: e2Payload},
				},
			},
		}
		app := setupAuthenticatedApp(t, ts)

		req := httptest.NewRequest(http.MethodGet, "/service/web/agents/list", http.NoBody)
		req.Header.Set("Cookie", "accessToken=test-token")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		text := string(body)
		assert.Contains(t, text, "Timed task")
		assert.Contains(t, text, "Total 4.6s")
		assert.Contains(t, text, `data-testid="chatagent-session-duration"`)
	})
}

func mustChatSessionEntryPayload(t *testing.T, entry session.TreeEntry) map[string]any {
	t.Helper()
	raw, err := session.MarshalEntry(entry)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &payload))
	return payload
}

func TestAgentsCreateSession(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		wantStatus int
	}{
		{name: "creates session when enabled", enabled: true, wantStatus: http.StatusCreated},
		{name: "disabled returns service unavailable", enabled: false, wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := pkgconfig.App.ChatAgent.ChatModel
			if tt.enabled {
				pkgconfig.App.ChatAgent.ChatModel = "test-model"
			} else {
				pkgconfig.App.ChatAgent.ChatModel = ""
			}
			t.Cleanup(func() { pkgconfig.App.ChatAgent.ChatModel = orig })

			ts := &testStore{}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodPost, "/service/web/agents", http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus == http.StatusCreated {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), "session_id")
				assert.NotEmpty(t, ts.chatSessions)
			}
		})
	}
}

func withChatAgentContextConfig(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# rules"), 0o644))
	model.RegisterTestMetadata(t, model.Metadata{
		ID:            "test-model",
		Name:          "Test Model",
		ContextLength: 100_000,
	})
	pkgconfig.App.ChatAgent.Workspace = root
}

func TestAgentChatContext(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		path       string
		auth       bool
		sessions   map[string]*gen.ChatSession
		wantStatus int
		wantBody   string
		checkJSON  func(t *testing.T, body []byte)
	}{
		{
			name:       "unauthenticated redirects to login",
			path:       "/service/web/agents/sess-owned/context",
			auth:       false,
			wantStatus: http.StatusSeeOther,
		},
		{
			name: "non-owner forbidden",
			path: "/service/web/agents/sess-other/context",
			auth: true,
			sessions: map[string]*gen.ChatSession{
				"sess-other": {Flag: "sess-other", UID: "someone-else", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "forbidden",
		},
		{
			name: "owner receives context breakdown json",
			path: "/service/web/agents/sess-owned/context",
			auth: true,
			sessions: map[string]*gen.ChatSession{
				"sess-owned": {Flag: "sess-owned", Title: "Owned", UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantStatus: http.StatusOK,
			checkJSON: func(t *testing.T, body []byte) {
				t.Helper()
				var report chatagent.ContextUsageReport
				require.NoError(t, json.Unmarshal(body, &report))
				assert.Equal(t, "test-model", report.Model)
				assert.Equal(t, 100_000, report.ContextWindow)
				gotIDs := make([]string, 0, len(report.Categories))
				for _, cat := range report.Categories {
					gotIDs = append(gotIDs, cat.ID)
				}
				assert.Contains(t, gotIDs, "system_prompt")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				withChatAgentContextConfig(t)
				ts := &testStore{chatSessionsByFlag: tt.sessions}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
				if tt.auth {
					req.Header.Set("Cookie", "accessToken=test-token")
				}
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				if tt.wantBody != "" {
					assert.Contains(t, string(body), tt.wantBody)
				}
				if tt.checkJSON != nil {
					tt.checkJSON(t, body)
				}
			})
		})
	}
}

func TestAgentChatPageOwner(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		path       string
		sessions   map[string]*gen.ChatSession
		wantStatus int
		wantBody   string
	}{
		{
			name: "owner can open chat page",
			path: "/service/web/agents/sess-owned",
			sessions: map[string]*gen.ChatSession{
				"sess-owned": {Flag: "sess-owned", Title: "Owned", UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantStatus: http.StatusOK,
			wantBody:   "chatagent-thread",
		},
		{
			name: "owner page includes context ring",
			path: "/service/web/agents/sess-owned",
			sessions: map[string]*gen.ChatSession{
				"sess-owned": {Flag: "sess-owned", Title: "Owned", UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantStatus: http.StatusOK,
			wantBody:   "chatagent-context-ring",
		},
		{
			name: "non-owner forbidden",
			path: "/service/web/agents/sess-other",
			sessions: map[string]*gen.ChatSession{
				"sess-other": {Flag: "sess-other", UID: "someone-else", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "forbidden",
		},
		{
			name:       "missing session not found",
			path:       "/service/web/agents/missing",
			sessions:   map[string]*gen.ChatSession{},
			wantStatus: http.StatusNotFound,
			wantBody:   "session not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{chatSessionsByFlag: tt.sessions}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
				req.Header.Set("Cookie", "accessToken=test-token")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentChatSendMessageValidation(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		body       string
		inFlight   bool
		wantStatus int
		wantBody   string
	}{
		{name: "empty message rejected", body: `{"text":"   "}`, wantStatus: http.StatusBadRequest, wantBody: "empty message"},
		{name: "invalid json rejected", body: `{`, wantStatus: http.StatusBadRequest, wantBody: "invalid json"},
		{name: "in flight returns conflict", body: `{"text":"hi"}`, inFlight: true, wantStatus: http.StatusConflict, wantBody: "run already in progress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				sessionID := "sess-msg"
				if tt.inFlight {
					pub := chatagent.NewChannelPublisher(4)
					gate := chatagent.NewConfirmGate(sessionID, pub)
					require.NoError(t, chatagent.TrySetAPIRunState(sessionID, chatagent.NewAPIRunState(pub, gate)))
					t.Cleanup(func() { chatagent.ClearAPIRunState(sessionID, nil) })
				}

				ts := &testStore{chatSessionsByFlag: map[string]*gen.ChatSession{
					sessionID: {Flag: sessionID, UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
				}}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodPost, "/service/web/agents/"+sessionID+"/messages", strings.NewReader(tt.body))
				req.Header.Set("Cookie", "accessToken=test-token")
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentRenderMarkdown(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "renders gfm table",
			body:       `{"text":"| A | B |\n| --- | --- |\n| 1 | 2 |"}`,
			wantStatus: http.StatusOK,
			wantBody:   "table",
		},
		{
			name:       "empty text rejected",
			body:       `{"text":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "empty text",
		},
		{
			name:       "invalid json rejected",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				app := setupAuthenticatedApp(t, &testStore{})
				req := httptest.NewRequest(http.MethodPost, "/service/web/agents/render-markdown", strings.NewReader(tt.body))
				req.Header.Set("Cookie", "accessToken=test-token")
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentsWebserviceStaticRoutesBeforeParam(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "list is static", path: "/agents/list"},
		{name: "render-markdown is static", path: "/agents/render-markdown"},
		{name: "create is collection", path: "/agents"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramIdx := -1
			staticIdx := -1
			for i, rule := range agentsWebserviceRules {
				if rule.Path == "/agents/:id" || strings.HasPrefix(rule.Path, "/agents/:id/") {
					if paramIdx < 0 {
						paramIdx = i
					}
				}
				if rule.Path == tt.path {
					staticIdx = i
				}
			}
			require.GreaterOrEqual(t, staticIdx, 0, "static route %s missing", tt.path)
			require.GreaterOrEqual(t, paramIdx, 0, "param route missing")
			assert.Less(t, staticIdx, paramIdx, "static route %s must be registered before /agents/:id", tt.path)
		})
	}
}

func TestAgentChatPageKeepsSessionIDOutOfRenderMarkdownPath(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name      string
		sessionID string
		wantURL   string
		wantNot   string
	}{
		{
			name:      "session suffix does not merge into render-markdown",
			sessionID: "CevXBUi8KZLY4oW7Hi4BPL",
			wantURL:   `data-messages-url="/service/web/agents/CevXBUi8KZLY4oW7Hi4BPL/messages"`,
			wantNot:   "render-markdown7Hi4BPL",
		},
		{
			name:      "render markdown url stays static",
			sessionID: "Wfm2Fx4vSzBz9z3Cbt2eeZ",
			wantURL:   `data-render-markdown-url="/service/web/agents/render-markdown"`,
			wantNot:   "render-markdownWfm2",
		},
		{
			name:      "session id attribute matches flag",
			sessionID: "7buyVEgKt25FBzqBAsRTPp",
			wantURL:   `data-session-id="7buyVEgKt25FBzqBAsRTPp"`,
			wantNot:   "render-markdown7buy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{chatSessionsByFlag: map[string]*gen.ChatSession{
					tt.sessionID: {
						Flag:      tt.sessionID,
						UID:       "testuser",
						State:     int(schema.ChatSessionActive),
						UpdatedAt: now,
						CreatedAt: now,
					},
				}}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, "/service/web/agents/"+tt.sessionID, http.NoBody)
				req.Header.Set("Cookie", "accessToken=test-token")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				html := string(body)
				assert.Contains(t, html, tt.wantURL)
				assert.NotContains(t, html, tt.wantNot)
			})
		})
	}
}

func TestMapChatMessages(t *testing.T) {
	tests := []struct {
		name string
		in   []chatagent.HistoryMessage
		want int
	}{
		{name: "maps user message", in: []chatagent.HistoryMessage{{Role: "user", Kind: "user", Text: "hi"}}, want: 1},
		{name: "maps assistant html", in: []chatagent.HistoryMessage{{Role: "assistant", Kind: "assistant", Text: "**bold**"}}, want: 1},
		{name: "persisted tool row", in: []chatagent.HistoryMessage{{Kind: "tool", Role: "tool", ToolName: "echo", ToolStatus: "completed", DurationMs: 50, Text: "ok"}}, want: 1},
		{name: "legacy assistant splits tool payload", in: []chatagent.HistoryMessage{{Role: "assistant", Text: "run_terminal({\"cmd\":\"ls\"})"}}, want: 1},
		{name: "empty input", in: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := mapChatMessages(tt.in)
			assert.Len(t, out, tt.want)
			if tt.want > 0 && tt.in[0].Role == "assistant" && tt.in[0].Kind != "tool" && tt.name != "legacy assistant splits tool payload" {
				assert.NotEmpty(t, out[0].HTML)
			}
			if tt.name == "persisted tool row" {
				assert.Equal(t, int64(50), out[0].DurationMs)
				assert.Equal(t, "echo", out[0].ToolName)
			}
		})
	}
}

func TestAgentChatConfirmNotFound(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		body       func(gateID string) string
		inFlight   bool
		resolve    bool
		wantStatus int
		wantBody   string
	}{
		{
			name: "missing gate returns not found",
			body: func(string) string {
				return `{"id":"stale-confirm","approved":true,"mode":"once"}`
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "confirm not found",
		},
		{
			name: "stale confirm id returns not found",
			body: func(string) string {
				return `{"id":"stale-confirm","approved":true,"mode":"once"}`
			},
			inFlight:   true,
			wantStatus: http.StatusNotFound,
			wantBody:   "confirm not found",
		},
		{
			name: "already resolved returns conflict",
			body: func(gateID string) string {
				return `{"id":"` + gateID + `","approved":true,"mode":"once"}`
			},
			inFlight:   true,
			resolve:    true,
			wantStatus: http.StatusConflict,
			wantBody:   "confirm already resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				sessionID := "sess-confirm"
				var gateID string
				if tt.inFlight {
					pub := chatagent.NewChannelPublisher(4)
					gate := chatagent.NewConfirmGate(sessionID, pub)
					gateID = gate.ID()
					require.NoError(t, chatagent.TrySetAPIRunState(sessionID, chatagent.NewAPIRunState(pub, gate)))
					t.Cleanup(func() { chatagent.ClearAPIRunState(sessionID, nil) })
					if tt.resolve {
						_, err := chatagent.ResolveConfirm(sessionID, gateID, true, chatagent.ConfirmModeOnce, "", chatagent.ConfirmReasonApproved)
						require.NoError(t, err)
					}
				}

				ts := &testStore{chatSessionsByFlag: map[string]*gen.ChatSession{
					sessionID: {Flag: sessionID, UID: "testuser", State: int(schema.ChatSessionActive), UpdatedAt: now, CreatedAt: now},
				}}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodPost, "/service/web/agents/"+sessionID+"/confirm", strings.NewReader(tt.body(gateID)))
				req.Header.Set("Cookie", "accessToken=test-token")
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentsCreateReturnsJSON(t *testing.T) {
	withChatAgentEnabled(t, func() {
		ts := &testStore{}
		app := setupAuthenticatedApp(t, ts)
		req := httptest.NewRequest(http.MethodPost, "/service/web/agents", bytes.NewReader(nil))
		req.Header.Set("Cookie", "accessToken=test-token")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	})
}
