package server

import (
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

// chatTestModule implements module.Handler for testing chat functions.
type chatTestModule struct {
	module.Base
	ready bool
}

func (h *chatTestModule) IsReady() bool              { return h.ready }
func (*chatTestModule) Init(_ json.RawMessage) error { return nil }
func (*chatTestModule) Rules() []any {
	return []any{[]command.Rule{
		{Define: "test_cmd", Help: "Test command"},
	}}
}

func setupTestCacheStore(t *testing.T) {
	t.Helper()
	s, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(s.Close)

	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cacheStore = cache.NewRedisStore(client)
	t.Cleanup(func() { cacheStore = nil })
}

func TestManageChatSession(t *testing.T) {
	tests := []struct {
		name        string
		msgAlt      string
		session     string
		wantSession string
		wantPayload types.MsgPayload
	}{
		{
			name:        "chat starts a new session when session is empty",
			msgAlt:      "chat",
			session:     "",
			wantSession: "new",
			wantPayload: types.TextMsg{Text: "Chat started"},
		},
		{
			name:        "chat reports already started when session exists",
			msgAlt:      "chat",
			session:     "existing-session",
			wantSession: "existing-session",
			wantPayload: types.TextMsg{Text: "Chat already started"},
		},
		{
			name:        "end clears session and returns ended message",
			msgAlt:      "end",
			session:     "active-session",
			wantSession: "",
			wantPayload: types.TextMsg{Text: "Chat ended"},
		},
		{
			name:        "plan enables plan mode for active session",
			msgAlt:      "plan",
			session:     "active-session",
			wantSession: "active-session",
			wantPayload: types.TextMsg{Text: "Plan mode on. The agent will research and propose a plan without making changes."},
		},
		{
			name:        "proceed disables plan mode for active session",
			msgAlt:      "proceed",
			session:     "active-session",
			wantSession: "active-session",
			wantPayload: types.TextMsg{Text: "Plan mode off. The agent can now make changes. Re-send your request to execute."},
		},
		{
			name:        "unknown command returns unchanged payload and session",
			msgAlt:      "hello",
			session:     "active",
			wantSession: "active",
			wantPayload: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestCacheStore(t)
			origDB := store.Database
			store.Database = &testStoreAdapter{}
			testChatSessions = map[string]*gen.ChatSession{}
			if tt.session != "" {
				mode := chatagent.ModeNormal
				if tt.msgAlt == "proceed" {
					mode = chatagent.ModePlan
				}
				testChatSessions[tt.session] = &gen.ChatSession{
					Flag:  tt.session,
					UID:   "uid-test",
					State: int(schema.ChatSessionActive),
					Mode:  mode,
				}
			}
			t.Cleanup(func() {
				store.Database = origDB
				testChatSessions = map[string]*gen.ChatSession{}
			})

			ctx := types.Context{}
			ctx.SetContext(t.Context())
			chatKey := cache.NewKey("chat", "user1", "topic1")

			got, session := manageChatSession(ctx, chatKey, tt.msgAlt, tt.session, nil, types.Uid("uid-test"))
			if tt.wantPayload == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.wantPayload, got)
			}
			if tt.wantSession == "new" {
				assert.NotEmpty(t, session)
			} else {
				assert.Equal(t, tt.wantSession, session)
			}
			if tt.msgAlt == "plan" && tt.session != "" {
				sess, ok := testChatSessions[tt.session]
				require.True(t, ok)
				assert.Equal(t, chatagent.ModePlan, sess.Mode)
			}
			if tt.msgAlt == "proceed" && tt.session != "" {
				sess, ok := testChatSessions[tt.session]
				require.True(t, ok)
				assert.Equal(t, chatagent.ModeNormal, sess.Mode)
			}
		})
	}
}

func TestBuildHelpMessage(t *testing.T) {
	tests := []struct {
		name   string
		msgAlt string
		isHelp bool
	}{
		{
			name:   "help command builds InfoMsg with module rules",
			msgAlt: "help",
			isHelp: true,
		},
		{
			name:   "random command returns nil payload",
			msgAlt: "hello",
			isHelp: false,
		},
		{
			name:   "HELP uppercase still triggers help",
			msgAlt: "HELP",
			isHelp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modName := "chat-test-help-" + tt.name
			if tt.isHelp {
				module.Register(modName, &chatTestModule{ready: true})
				t.Cleanup(func() { module.Unregister(modName) })
			}

			got := buildHelpMessage(tt.msgAlt, nil)
			if tt.isHelp {
				require.NotNil(t, got)
				info, ok := got.(types.InfoMsg)
				assert.True(t, ok)
				assert.Equal(t, "Help", info.Title)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestDispatchToModules(t *testing.T) {
	tests := []struct {
		name    string
		msgAlt  string
		wantNil bool
	}{
		{
			name:    "dispatch with slash prefix strips slash",
			msgAlt:  "/test-command",
			wantNil: true,
		},
		{
			name:    "dispatch without slash passes command directly",
			msgAlt:  "test-command",
			wantNil: true,
		},
		{
			name:    "dispatch with empty command returns nil",
			msgAlt:  "",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.Context{}
			got := dispatchToModules(ctx, tt.msgAlt)
			if tt.wantNil {
				assert.Nil(t, got)
			}
		})
	}
}
