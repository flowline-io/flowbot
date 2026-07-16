package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestChatAgentHTTPSessionEventsObserverFilter(t *testing.T) {
	origDB := store.Database
	origCfg := config.App.ChatAgent
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-ev": {Flag: "sess-ev", UID: "user-1", State: int(schema.ChatSessionActive)},
	}
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	t.Cleanup(func() {
		store.Database = origDB
		config.App.ChatAgent = origCfg
		testChatSessions = map[string]*gen.ChatSession{}
		chatagent.ResetSessionEventHubsForTest()
	})
	chatagent.ResetSessionEventHubsForTest()

	h := newChatAgentHTTP()
	app := fiber.New()
	app.Get("/chatagent/sessions/:id/events", func(c fiber.Ctx) error {
		c.Locals("route:ctx", &route.RequestContext{
			UID:    types.Uid("user-1"),
			Scopes: []string{auth.ScopeChatAgentChat},
		})
		return h.sessionEvents(c)
	})

	tests := []struct {
		name          string
		events        []chatagent.StreamEvent
		wantInBody    []string
		wantNotInBody []string
	}{
		{
			name: "forwards confirm and drops delta",
			events: []chatagent.StreamEvent{
				{Type: chatagent.EventTypeDelta, Text: "stream-secret"},
				{Type: chatagent.EventTypeConfirm, ID: "c-1", Tool: "bash"},
			},
			wantInBody:    []string{`"type":"confirm"`, `"id":"c-1"`},
			wantNotInBody: []string{`"type":"delta"`, "stream-secret"},
		},
		{
			name: "forwards mode_change and drops tool",
			events: []chatagent.StreamEvent{
				{Type: chatagent.EventTypeTool, Name: "read_file"},
				{Type: chatagent.EventTypeModeChange, Mode: chatagent.ModePlan},
			},
			wantInBody:    []string{`"type":"mode_change"`, chatagent.ModePlan},
			wantNotInBody: []string{`"type":"tool"`},
		},
		{
			name: "forwards canceled and drops done",
			events: []chatagent.StreamEvent{
				{Type: chatagent.EventTypeDone},
				{Type: chatagent.EventTypeCanceled, Message: "stopped"},
			},
			wantInBody:    []string{`"type":"canceled"`},
			wantNotInBody: []string{`"type":"done"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatagent.ResetSessionEventHubsForTest()

			req := httptest.NewRequest(http.MethodGet, "/chatagent/sessions/sess-ev/events", http.NoBody)

			type result struct {
				body []byte
				err  error
			}
			done := make(chan result, 1)
			go func() {
				resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
				if err != nil {
					done <- result{err: err}
					return
				}
				body, readErr := io.ReadAll(resp.Body)
				done <- result{body: body, err: readErr}
			}()

			time.Sleep(100 * time.Millisecond)
			for _, ev := range tt.events {
				chatagent.PublishSessionEvent("sess-ev", ev)
			}
			// Non-terminal observer events keep the SSE stream open; canceled ends it
			// (same as production clients disconnecting). req.Context cancel is unreliable
			// under fiber app.Test for long-lived streams.
			chatagent.PublishSessionEvent("sess-ev", chatagent.StreamEvent{
				Type:    chatagent.EventTypeCanceled,
				Message: "test done",
			})
			time.Sleep(50 * time.Millisecond)

			var got result
			select {
			case got = <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("session events request did not finish")
			}
			require.NoError(t, got.err)

			text := string(got.body)
			for _, want := range tt.wantInBody {
				assert.Contains(t, text, want)
			}
			for _, not := range tt.wantNotInBody {
				assert.NotContains(t, text, not)
			}
		})
	}
}
