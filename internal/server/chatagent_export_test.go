package server

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatAgentExportSession(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		setup     func()
		wantCount int
		wantErr   bool
	}{
		{
			name: "exports all entries",
			setup: func() {
				testChatSessions["sess-1"] = &gen.ChatSession{
					Flag: "sess-1", UID: "user-1", LeafID: "e2",
					State: int(schema.ChatSessionActive), CreatedAt: now, UpdatedAt: now,
				}
				testChatSessionEntries["sess-1"] = []*gen.ChatSessionEntry{
					{
						Flag: "e1", SessionID: "sess-1", ParentID: "", EntryType: "message",
						Payload: mustChatAgentExportEntry(t, session.TreeEntry{
							ID: "e1", Type: session.EntryMessage,
							Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
						}),
					},
					{
						Flag: "e2", SessionID: "sess-1", ParentID: "e1", EntryType: "compaction",
						Payload: mustChatAgentExportEntry(t, session.TreeEntry{
							ID: "e2", ParentID: "e1", Type: session.EntryCompaction, Summary: "compact",
						}),
					},
				}
			},
			wantCount: 2,
		},
		{
			name: "empty session",
			setup: func() {
				testChatSessions["sess-1"] = &gen.ChatSession{
					Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}
			},
			wantCount: 0,
		},
		{
			name:    "missing session",
			setup:   func() {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origDB := store.Database
			origCfg := config.App.ChatAgent
			store.Database = &testStoreAdapter{}
			testChatSessions = map[string]*gen.ChatSession{}
			testChatSessionEntries = map[string][]*gen.ChatSessionEntry{}
			config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
			t.Cleanup(func() {
				store.Database = origDB
				config.App.ChatAgent = origCfg
				testChatSessions = map[string]*gen.ChatSession{}
				testChatSessionEntries = map[string][]*gen.ChatSessionEntry{}
			})

			tt.setup()
			export, err := chatagent.ExportSession(context.Background(), "sess-1")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, export)
			assert.Equal(t, "sess-1", export.SessionID)
			assert.Equal(t, tt.wantCount, export.EntryCount)
			assert.Len(t, export.Entries, tt.wantCount)
		})
	}
}

func TestChatAgentHTTPExportSession(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", UID: "user-1", State: int(schema.ChatSessionActive), CreatedAt: now, UpdatedAt: now},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		testChatSessionEntries = map[string][]*gen.ChatSessionEntry{}
	})

	h := newChatAgentHTTP()
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/export", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.exportSession(c)
	})

	req := httptest.NewRequest("GET", "/chatagent/sessions/sess-1/export", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var parsed chatagent.SessionExport
	require.NoError(t, sonic.Unmarshal(body, &parsed))
	assert.Equal(t, "sess-1", parsed.SessionID)
}

func mustChatAgentExportEntry(t *testing.T, entry session.TreeEntry) map[string]any {
	t.Helper()
	data, err := session.MarshalEntry(entry)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, sonic.Unmarshal(data, &payload))
	return payload
}
