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
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanModePermissionHook(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-plan": {
			Flag:  "sess-plan",
			UID:   "user-1",
			State: int(schema.ChatSessionActive),
			Mode:  chatagent.ModePlan,
		},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		chatagent.ResetPermissionCacheForTest()
		ChatAgentService().ResetPermissionSessionsForTest()
	})

	tests := []struct {
		name    string
		tool    string
		wantBlk bool
	}{
		{name: "write_file blocked in plan mode", tool: permission.ToolWriteFile, wantBlk: true},
		{name: "run_terminal blocked in plan mode", tool: permission.ToolRunTerminal, wantBlk: true},
		{name: "run_code blocked in plan mode", tool: permission.ToolRunCode, wantBlk: true},
		{name: "read_file allowed in plan mode", tool: permission.ToolReadFile, wantBlk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := hooks.NewRegistry()
			chatagent.RegisterHooks(reg, chatagent.ChatHookDeps{
				SessionID:   "sess-plan",
				UID:         types.Uid("user-1"),
				SessionMode: chatagent.ModePlan,
				DCG:         dcg.AllowAllChecker{},
				Service:     ChatAgentService(),
			})
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
				Args: map[string]any{
					"path":     "README.md",
					"command":  "ls",
					"language": "python",
					"code":     "print(1)",
				},
			})
			require.NoError(t, err)
			if tt.wantBlk {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				assert.Equal(t, "plan mode: read-only", result.Reason)
				return
			}
			if result != nil {
				assert.False(t, result.Block)
			}
		})
	}
}

func TestChatAgentHTTPSessionMode(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {
			Flag:  "sess-1",
			UID:   "user-1",
			State: int(schema.ChatSessionActive),
			Mode:  chatagent.ModeNormal,
		},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
	})

	h := newChatAgentHTTP(ChatAgentService())
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/mode", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.getSessionMode(c)
	})
	app.Put("/chatagent/sessions/:id/mode", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.putSessionMode(c)
	})

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantMode   string
	}{
		{name: "get defaults to normal", method: http.MethodGet, wantStatus: http.StatusOK, wantMode: chatagent.ModeNormal},
		{name: "put plan mode", method: http.MethodPut, body: `{"mode":"plan"}`, wantStatus: http.StatusOK, wantMode: chatagent.ModePlan},
		{name: "put invalid mode", method: http.MethodPut, body: `{"mode":"debug"}`, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodGet {
				req = httptest.NewRequest(http.MethodGet, "/chatagent/sessions/sess-1/mode", http.NoBody)
			} else {
				req = httptest.NewRequest(http.MethodPut, "/chatagent/sessions/sess-1/mode", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantMode == "" {
				return
			}
			body, readErr := io.ReadAll(resp.Body)
			require.NoError(t, readErr)
			var parsed map[string]string
			require.NoError(t, sonic.Unmarshal(body, &parsed))
			assert.Equal(t, tt.wantMode, parsed["mode"])
			assert.Equal(t, tt.wantMode, testChatSessions["sess-1"].Mode)
		})
	}
}

func TestSetSessionModeAndNotify(t *testing.T) {
	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {
			Flag:  "sess-1",
			UID:   "user-1",
			State: int(schema.ChatSessionActive),
			Mode:  chatagent.ModeNormal,
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		testChatSessions = map[string]*gen.ChatSession{}
		ChatAgentService().ResetSessionEventHubsForTest()
	})
	ChatAgentService().ResetSessionEventHubsForTest()

	hub := ChatAgentService().GetSessionEventHub("sess-1")
	pub := hub.Subscribe("test", 4)
	t.Cleanup(func() { hub.Unsubscribe("test") })

	ctx := context.Background()
	require.NoError(t, ChatAgentService().SetSessionModeAndNotify(ctx, "sess-1", chatagent.ModePlan))

	select {
	case ev := <-pub.Events():
		assert.Equal(t, chatagent.EventTypeModeChange, ev.Type)
		assert.Equal(t, chatagent.ModePlan, ev.Mode)
	case <-time.After(time.Second):
		t.Fatal("expected mode_change event")
	}
	assert.Equal(t, chatagent.ModePlan, testChatSessions["sess-1"].Mode)
}
