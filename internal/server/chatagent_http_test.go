package server

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

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

	req := httptest.NewRequest("GET", "/chatagent/info", nil)
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

	req := httptest.NewRequest("POST", "/chatagent/sessions", nil)
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

	req := httptest.NewRequest("GET", "/chatagent/sessions/sess-1/messages", nil)
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
