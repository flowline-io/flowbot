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
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScheduledHTTPTest(t *testing.T) (*chatAgentHTTP, *fiber.App, types.Uid) {
	t.Helper()
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = postgres.NewSQLiteTestAdapter(t)
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
	})

	h := newChatAgentHTTP()
	app := fiber.New()
	uid := types.Uid("user:alice")
	wrap := func(fn func(fiber.Ctx) error) fiber.Handler {
		return func(c fiber.Ctx) error {
			c.Locals("route:ctx", &route.RequestContext{UID: uid, Scopes: []string{auth.ScopeChatAgentChat}})
			return fn(c)
		}
	}
	app.Get("/chatagent/scheduled-tasks", wrap(h.listScheduledTasks))
	app.Post("/chatagent/scheduled-tasks", wrap(h.createScheduledTask))
	app.Get("/chatagent/scheduled-tasks/:id", wrap(h.getScheduledTask))
	app.Patch("/chatagent/scheduled-tasks/:id", wrap(h.patchScheduledTask))
	app.Delete("/chatagent/scheduled-tasks/:id", wrap(h.cancelScheduledTask))
	app.Get("/chatagent/scheduled-tasks/:id/runs", wrap(h.listScheduledTaskRuns))
	return h, app, uid
}

func TestChatAgentHTTPScheduledTasksCRUD(t *testing.T) {
	_, app, uid := setupScheduledHTTPTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, store.Database.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
		Flag: "task-http-1", UID: uid.String(), Name: "daily",
		ScheduleKind: string(schema.ChatScheduledTaskKindCron),
		Cron:         "0 9 * * *", Prompt: "check logs",
		State: string(schema.ChatScheduledTaskStateActive), CreatedAt: now, UpdatedAt: now,
	}))

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantSub    string
	}{
		{name: "list tasks", method: http.MethodGet, path: "/chatagent/scheduled-tasks", wantStatus: fiber.StatusOK, wantSub: "task-http-1"},
		{name: "get task", method: http.MethodGet, path: "/chatagent/scheduled-tasks/task-http-1", wantStatus: fiber.StatusOK, wantSub: "daily"},
		{name: "patch task prompt", method: http.MethodPatch, path: "/chatagent/scheduled-tasks/task-http-1", body: `{"prompt":"updated"}`, wantStatus: fiber.StatusOK, wantSub: "updated"},
		{name: "list runs empty", method: http.MethodGet, path: "/chatagent/scheduled-tasks/task-http-1/runs", wantStatus: fiber.StatusOK, wantSub: "runs"},
		{name: "cancel task", method: http.MethodDelete, path: "/chatagent/scheduled-tasks/task-http-1", wantStatus: fiber.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, http.NoBody)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantSub != "" && resp.StatusCode != fiber.StatusNoContent {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.Contains(t, string(body), tt.wantSub)
			}
		})
	}
}

func TestChatAgentHTTPCreateScheduledTask(t *testing.T) {
	_, app, _ := setupScheduledHTTPTest(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantSub    string
	}{
		{
			name:       "creates cron task",
			body:       `{"name":"weekly","schedule_kind":"cron","cron":"0 8 * * 1","prompt":"weekly report"}`,
			wantStatus: fiber.StatusCreated,
			wantSub:    "weekly",
		},
		{
			name:       "invalid json rejected",
			body:       `{bad`,
			wantStatus: fiber.StatusBadRequest,
			wantSub:    "invalid json",
		},
		{
			name:       "missing prompt rejected",
			body:       `{"name":"x","schedule_kind":"cron","cron":"0 8 * * 1"}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/chatagent/scheduled-tasks", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantSub != "" {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.Contains(t, string(body), tt.wantSub)
			}
		})
	}
}

func TestChatAgentHTTPScheduledUnauthorized(t *testing.T) {
	origCfg := config.App.ChatAgent
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() { config.App.ChatAgent = origCfg })

	h := newChatAgentHTTP()
	app := fiber.New()
	app.Get("/chatagent/scheduled-tasks", h.listScheduledTasks)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/chatagent/scheduled-tasks", http.NoBody))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestChatAgentHTTPScheduledInvalidLimit(t *testing.T) {
	_, app, uid := setupScheduledHTTPTest(t)
	ctx := context.Background()
	require.NoError(t, store.Database.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
		Flag: "task-limit", UID: uid.String(), Name: "daily",
		ScheduleKind: string(schema.ChatScheduledTaskKindCron),
		Cron:         "0 9 * * *", Prompt: "check",
		State: string(schema.ChatScheduledTaskStateActive),
	}))

	req := httptest.NewRequest(http.MethodGet, "/chatagent/scheduled-tasks/task-limit/runs?limit=abc", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var parsed map[string]string
	require.NoError(t, sonic.Unmarshal(body, &parsed))
	assert.Equal(t, "invalid limit", parsed["error"])
}
