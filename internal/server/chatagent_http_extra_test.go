package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestChatAgentHTTPGetSessionAndExport(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-export": {Flag: "sess-export", UID: "user-1", State: int(schema.ChatSessionActive), Title: "Deploy"},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP()
	app := fiber.New()
	wrap := func(fn func(fiber.Ctx) error) fiber.Handler {
		return func(c fiber.Ctx) error {
			c.Locals("route:ctx", &route.RequestContext{
				UID: types.Uid("user-1"), Scopes: []string{auth.ScopeChatAgentChat},
			})
			return fn(c)
		}
	}
	app.Get("/chatagent/sessions/:id/export", wrap(h.exportSession))
	app.Get("/chatagent/sessions", wrap(h.listSessions))

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "list sessions", path: "/chatagent/sessions", wantStatus: fiber.StatusOK},
		{name: "export session", path: "/chatagent/sessions/sess-export/export", wantStatus: fiber.StatusOK},
		{name: "missing session export", path: "/chatagent/sessions/missing/export", wantStatus: fiber.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestRequireChatAgentEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		wantErr bool
	}{
		{name: "disabled returns error", enabled: false, wantErr: true},
		{name: "enabled passes", enabled: true, wantErr: false},
		{name: "model only still disabled without workspace", enabled: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := config.App.ChatAgent
			if tt.enabled {
				config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
			} else {
				config.App.ChatAgent = config.ChatAgentConfig{}
			}
			t.Cleanup(func() { config.App.ChatAgent = orig })

			err := requireChatAgentEnabled()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildPollingStateWithDatabase(t *testing.T) {
	origDB := store.Database
	store.Database = &testStoreAdapter{}
	t.Cleanup(func() { store.Database = origDB })

	state := buildPollingState()
	require.NotNil(t, state)
}
