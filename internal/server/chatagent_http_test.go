package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatAgentHTTPDisabled(t *testing.T) {
	orig := config.App.ChatAgent
	config.App.ChatAgent = config.ChatAgentConfig{}
	t.Cleanup(func() { config.App.ChatAgent = orig })

	app := fiber.New()
	app.Get("/chatagent/info", newChatAgentHTTP(ChatAgentService()).info)

	req := httptest.NewRequest("GET", "/chatagent/info", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestChatAgentHTTPCreateSession(t *testing.T) {
	origCfg := config.App
	setupSQLiteTestDB(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-test", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Post("/chatagent/sessions", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.createSession(c)
	})

	tests := []struct {
		name       string
		body       string
		wantModel  string
		wantLevel  string
		wantStatus int
	}{
		{
			name:       "empty body creates session",
			body:       "",
			wantStatus: fiber.StatusCreated,
		},
		{
			name:       "stores model and thinking level",
			body:       `{"model":"gpt-alt","thinking_level":"high"}`,
			wantModel:  "gpt-alt",
			wantLevel:  "high",
			wantStatus: fiber.StatusCreated,
		},
		{
			name:       "rejects unknown model",
			body:       `{"model":"nope","thinking_level":"default"}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest("POST", "/chatagent/sessions", http.NoBody)
			} else {
				req = httptest.NewRequest("POST", "/chatagent/sessions", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus != fiber.StatusCreated {
				return
			}
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			var parsed map[string]string
			require.NoError(t, sonic.Unmarshal(raw, &parsed))
			require.NotEmpty(t, parsed["session_id"])
			if tt.wantModel == "" && tt.wantLevel == "" {
				return
			}
			sess := getTestChatSession(t, parsed["session_id"])
			assert.Equal(t, tt.wantModel, sess.Model)
			assert.Equal(t, tt.wantLevel, sess.ThinkingLevel)
		})
	}
}

func TestChatAgentHTTPSessionSettings(t *testing.T) {
	origCfg := config.App
	setupSQLiteTestDB(t)
	seedTestChatSession(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-test", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	withOwner := func(handler fiber.Handler) fiber.Handler {
		return func(c fiber.Ctx) error {
			c.Locals("route:ctx", &route.RequestContext{
				UID:    types.Uid("user-1"),
				Scopes: []string{auth.ScopeChatAgentChat},
			})
			return handler(c)
		}
	}
	app.Get("/chatagent/sessions/:id/settings", withOwner(h.getSessionSettings))
	app.Put("/chatagent/sessions/:id/settings", withOwner(h.putSessionSettings))

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantModel  string
		wantLevel  string
	}{
		{
			name:       "get empty settings",
			method:     http.MethodGet,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "put valid settings",
			method:     http.MethodPut,
			body:       `{"model":"gpt-alt","thinking_level":"medium"}`,
			wantStatus: fiber.StatusOK,
			wantModel:  "gpt-alt",
			wantLevel:  "medium",
		},
		{
			name:       "put invalid thinking level",
			method:     http.MethodPut,
			body:       `{"model":"gpt-alt","thinking_level":"turbo"}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodGet {
				req = httptest.NewRequest(tt.method, "/chatagent/sessions/sess-1/settings", http.NoBody)
			} else {
				req = httptest.NewRequest(tt.method, "/chatagent/sessions/sess-1/settings", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus != fiber.StatusOK || tt.method == http.MethodGet && tt.wantModel == "" {
				if tt.method == http.MethodPut && tt.wantStatus == fiber.StatusOK {
					sess := getTestChatSession(t, "sess-1")
					assert.Equal(t, tt.wantModel, sess.Model)
					assert.Equal(t, tt.wantLevel, sess.ThinkingLevel)
				}
				return
			}
			if tt.method == http.MethodPut {
				sess := getTestChatSession(t, "sess-1")
				assert.Equal(t, tt.wantModel, sess.Model)
				assert.Equal(t, tt.wantLevel, sess.ThinkingLevel)
			}
		})
	}
}

func TestChatAgentHTTPListMessages(t *testing.T) {
	setupChatAgentHTTPTest(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/messages", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.listMessages(c)
	})

	req := httptest.NewRequest("GET", "/chatagent/sessions/sess-1/messages", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestChatAgentHTTPListTrajectory(t *testing.T) {
	setupChatAgentHTTPTest(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/trajectory", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.listTrajectory(c)
	})

	req := httptest.NewRequest("GET", "/chatagent/sessions/sess-1/trajectory", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestChatAgentHTTPConfirmNotFound(t *testing.T) {
	setupChatAgentHTTPTest(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Post("/chatagent/sessions/:id/confirm", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{UID: types.Uid("user-1")})
		return h.confirm(c)
	})

	body := `{"id":"missing","approved":true}`
	req := httptest.NewRequest("POST", "/chatagent/sessions/sess-1/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestChatAgentHTTPEmptyMessage(t *testing.T) {
	setupChatAgentHTTPTest(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Post("/chatagent/sessions/:id/messages", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.sendMessage(c)
	})

	req := httptest.NewRequest("POST", "/chatagent/sessions/sess-1/messages", strings.NewReader(`{"text":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestChatAgentHTTPRunInFlight(t *testing.T) {
	setupChatAgentHTTPTest(t, &gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)})
	t.Cleanup(func() {
		ChatAgentService().ClearAPIRunState("sess-1", nil)
	})

	pub := chatagent.NewChannelPublisher(4)
	gate := chatagent.NewConfirmGate("sess-1", pub, nil)
	require.NoError(t, ChatAgentService().TrySetAPIRunState("sess-1", chatagent.NewAPIRunState(pub, gate)))

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Post("/chatagent/sessions/:id/messages", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.sendMessage(c)
	})

	req := httptest.NewRequest("POST", "/chatagent/sessions/sess-1/messages", strings.NewReader(`{"text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestChatAgentHTTPListSessions(t *testing.T) {
	now := time.Now().UTC()
	setupChatAgentHTTPTest(t,
		&gen.ChatSession{Flag: "sess-a", UID: "user-1", State: int(schema.ChatSessionActive), UpdatedAt: now},
		&gen.ChatSession{Flag: "sess-b", UID: "user-2", State: int(schema.ChatSessionActive), UpdatedAt: now},
		&gen.ChatSession{Flag: "sess-c", UID: "user-1", State: int(schema.ChatSessionClosed), UpdatedAt: now},
	)

	h := newChatAgentHTTP(ChatAgentService())

	tests := []struct {
		name       string
		uid        types.Uid
		query      string
		wantStatus int
		wantLen    int
	}{
		{
			name:       "returns active sessions for authenticated user",
			uid:        types.Uid("user-1"),
			wantStatus: fiber.StatusOK,
			wantLen:    1,
		},
		{
			name:       "unauthorized without uid",
			uid:        types.Uid(""),
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "invalid limit returns bad request",
			uid:        types.Uid("user-1"),
			query:      "?limit=bad",
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/chatagent/sessions", func(c fiber.Ctx) error {
				c.Locals("route:ctx", &route.RequestContext{
					UID:    tt.uid,
					Scopes: []string{auth.ScopeChatAgentChat},
				})
				return h.listSessions(c)
			})

			req := httptest.NewRequest("GET", "/chatagent/sessions"+tt.query, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantLen == 0 {
				return
			}
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			var parsed struct {
				Sessions []map[string]any `json:"sessions"`
			}
			require.NoError(t, sonic.Unmarshal(body, &parsed))
			assert.Len(t, parsed.Sessions, tt.wantLen)
		})
	}
}

func TestChatAgentHTTPGetPermissionsSessionOwner(t *testing.T) {
	setupChatAgentHTTPTest(t,
		&gen.ChatSession{Flag: "sess-mine", UID: "user-1", State: int(schema.ChatSessionActive)},
		&gen.ChatSession{Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	)
	t.Cleanup(func() {
		ChatAgentService().ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/permissions", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.getPermissions(c)
	})

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{name: "own session grants", query: "?session_id=sess-mine", wantStatus: fiber.StatusOK},
		{name: "foreign session forbidden", query: "?session_id=sess-other", wantStatus: fiber.StatusForbidden},
		{name: "no session id ok", query: "", wantStatus: fiber.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/chatagent/permissions"+tt.query, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestChatAgentHTTPPermissionsMutations(t *testing.T) {
	setupChatAgentHTTPTest(t)
	t.Cleanup(func() {
		chatagent.ResetPermissionCacheForTest()
		ChatAgentService().ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Put("/chatagent/permissions", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.putPermissions(c)
	})
	app.Delete("/chatagent/permissions", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.deletePermissions(c)
	})

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{name: "put valid permissions", method: http.MethodPut, body: `{"bash":{"default":"deny"}}`, wantStatus: fiber.StatusOK},
		{name: "put invalid permissions", method: http.MethodPut, body: `{"bash":"allow"}`, wantStatus: fiber.StatusBadRequest},
		{name: "delete permissions", method: http.MethodDelete, body: "", wantStatus: fiber.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader = http.NoBody
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, "/chatagent/permissions", body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestChatAgentHTTPClearPermissionGrants(t *testing.T) {
	setupChatAgentHTTPTest(t,
		&gen.ChatSession{Flag: "sess-grants", UID: "user-1", State: int(schema.ChatSessionActive)},
		&gen.ChatSession{Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	)
	t.Cleanup(func() {
		ChatAgentService().ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Delete("/chatagent/sessions/:id/permission-grants", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.clearPermissionGrants(c)
	})

	tests := []struct {
		name       string
		sessionID  string
		wantStatus int
	}{
		{name: "clears own session grants", sessionID: "sess-grants", wantStatus: fiber.StatusNoContent},
		{name: "foreign session forbidden", sessionID: "sess-other", wantStatus: fiber.StatusForbidden},
		{name: "missing session not found", sessionID: "missing", wantStatus: fiber.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/chatagent/sessions/"+tt.sessionID+"/permission-grants", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestChatAgentHTTPCancelRun(t *testing.T) {
	setupChatAgentHTTPTest(t,
		&gen.ChatSession{Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
		&gen.ChatSession{Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	)

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Post("/chatagent/sessions/:id/cancel", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.cancelRun(c)
	})

	tests := []struct {
		name       string
		sessionID  string
		wantStatus int
	}{
		{name: "cancel own session", sessionID: "sess-1", wantStatus: fiber.StatusNoContent},
		{name: "foreign session forbidden", sessionID: "sess-other", wantStatus: fiber.StatusForbidden},
		{name: "missing session not found", sessionID: "missing", wantStatus: fiber.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/chatagent/sessions/"+tt.sessionID+"/cancel", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestListUserActiveSessions(t *testing.T) {
	now := time.Now().UTC()
	setupSQLiteTestDB(t)
	seedTestChatSession(t, &gen.ChatSession{Flag: "sess-a", UID: "user-1", Title: "Redis setup", State: int(schema.ChatSessionActive), UpdatedAt: now})
	seedTestChatSession(t, &gen.ChatSession{Flag: "sess-b", UID: "user-2", State: int(schema.ChatSessionActive), UpdatedAt: now})
	seedTestChatSession(t, &gen.ChatSession{Flag: "sess-c", UID: "user-1", State: int(schema.ChatSessionClosed), UpdatedAt: now})

	tests := []struct {
		name    string
		uid     types.Uid
		setupDB bool
		wantLen int
		wantErr bool
	}{
		{name: "returns active sessions for uid", uid: types.Uid("user-1"), setupDB: true, wantLen: 1},
		{name: "empty result for other uid", uid: types.Uid("user-9"), setupDB: true, wantLen: 0},
		{name: "unavailable store returns error", uid: types.Uid("user-1"), setupDB: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.setupDB {
				orig := store.Database
				store.Database = nil
				t.Cleanup(func() { store.Database = orig })
			}
			got, _, err := chatagent.ListUserActiveSessions(context.Background(), tt.uid, 20, "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "sess-a", got[0].SessionID)
				assert.Equal(t, "Redis setup", got[0].Title)
				assert.Equal(t, "active", got[0].State)
			}
		})
	}
}
