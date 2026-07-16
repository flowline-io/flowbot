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
	app.Get("/chatagent/info", newChatAgentHTTP().info)

	req := httptest.NewRequest("GET", "/chatagent/info", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestChatAgentHTTPCreateSession(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP()
	app := fiber.New()
	app.Post("/chatagent/sessions", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.createSession(c)
	})

	req := httptest.NewRequest("POST", "/chatagent/sessions", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var parsed map[string]string
	require.NoError(t, sonic.Unmarshal(body, &parsed))
	assert.NotEmpty(t, parsed["session_id"])
}

func TestChatAgentHTTPListMessages(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP()
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

func TestChatAgentHTTPConfirmNotFound(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		chatagent.ClearAPIRunState("sess-1", nil)
	})

	pub := chatagent.NewChannelPublisher(4)
	gate := chatagent.NewConfirmGate("sess-1", pub)
	require.NoError(t, chatagent.TrySetAPIRunState("sess-1", chatagent.NewAPIRunState(pub, gate)))

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-a": {Flag: "sess-a", UID: "user-1", State: int(schema.ChatSessionActive), UpdatedAt: now},
		"sess-b": {Flag: "sess-b", UID: "user-2", State: int(schema.ChatSessionActive), UpdatedAt: now},
		"sess-c": {Flag: "sess-c", UID: "user-1", State: int(schema.ChatSessionClosed), UpdatedAt: now},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP()

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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-mine":  {Flag: "sess-mine", UID: "user-1", State: int(schema.ChatSessionActive)},
		"sess-other": {Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		chatagent.ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		chatagent.ResetPermissionCacheForTest()
		chatagent.ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-grants": {Flag: "sess-grants", UID: "user-1", State: int(schema.ChatSessionActive)},
		"sess-other":  {Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		chatagent.ResetPermissionSessionsForTest()
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1":     {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
		"sess-other": {Flag: "sess-other", UID: "user-2", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP()
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
	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-a": {Flag: "sess-a", UID: "user-1", Title: "Redis setup", State: int(schema.ChatSessionActive), UpdatedAt: now},
		"sess-b": {Flag: "sess-b", UID: "user-2", State: int(schema.ChatSessionActive), UpdatedAt: now},
		"sess-c": {Flag: "sess-c", UID: "user-1", State: int(schema.ChatSessionClosed), UpdatedAt: now},
	}
	t.Cleanup(func() {
		store.Database = origDB
		testChatSessions = map[string]*gen.ChatSession{}
	})

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
			if tt.setupDB {
				store.Database = &testStoreAdapter{}
			} else {
				store.Database = nil
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
