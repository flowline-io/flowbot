package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestChatAgentHTTPGetResourceRequiresSessionID(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/resources", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.getResource(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/chatagent/resources?uri=plan://p1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestChatAgentHTTPGetPlanResource(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	testAgentPlans = map[string]*gen.AgentPlan{
		"p1": {
			Flag:      "p1",
			SessionID: "sess-1",
			Title:     "Deploy",
			Content:   "# Plan",
			CreatedAt: time.Now().UTC(),
		},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		testAgentPlans = map[string]*gen.AgentPlan{}
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/resources", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.getResource(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/chatagent/resources?session_id=sess-1&uri=plan://p1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Deploy")
	assert.Contains(t, string(body), "# Plan")
}

func TestChatAgentHTTPListSessionPlans(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	now := time.Now().UTC()
	testAgentPlans = map[string]*gen.AgentPlan{
		"p1": {Flag: "p1", SessionID: "sess-1", Title: "A", CreatedAt: now},
		"p2": {Flag: "p2", SessionID: "sess-1", Title: "B", CreatedAt: now.Add(time.Minute)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		testAgentPlans = map[string]*gen.AgentPlan{}
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/plans", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.listSessionPlans(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/chatagent/sessions/sess-1/plans", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "plan://p1")
	assert.Contains(t, string(body), "plan://p2")
}

func TestChatAgentHTTPListSessionTodos(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	testAgentTodos = map[string]*gen.AgentTodo{
		"t1": {Flag: "t1", SessionID: "sess-1", ItemID: "a", Content: "Plan work", Status: "pending", SortOrder: 0},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		testAgentTodos = map[string]*gen.AgentTodo{}
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/todos", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.listSessionTodos(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/chatagent/sessions/sess-1/todos", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"item_id":"a"`)
	assert.Contains(t, string(body), "Plan work")
}
