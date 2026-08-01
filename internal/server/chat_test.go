package server

import (
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
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
			setupSQLiteTestDB(t)
			uid := types.Uid("uid-test")
			scope := "thread-1"
			ctx := types.Context{}
			ctx.SetContext(t.Context())
			if tt.session != "" {
				mode := chatagent.ModeNormal
				if tt.msgAlt == "proceed" {
					mode = chatagent.ModePlan
				}
				seedTestChatSession(t, &gen.ChatSession{
					Flag:  tt.session,
					UID:   uid.String(),
					State: int(schema.ChatSessionActive),
					Mode:  mode,
				})
				require.NoError(t, bindThreadSession(ctx, uid, scope, tt.session))
			}

			got, session := manageChatSession(ctx, uid, scope, tt.msgAlt, tt.session, nil)
			if tt.wantPayload == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.wantPayload, got)
			}
			if tt.wantSession == "new" {
				assert.NotEmpty(t, session)
				assert.True(t, isChatEnabled(ctx, uid))
			} else {
				assert.Equal(t, tt.wantSession, session)
			}
			if tt.msgAlt == "end" {
				assert.False(t, isChatEnabled(ctx, uid))
			}
			if tt.msgAlt == "plan" && tt.session != "" {
				sess := getTestChatSession(t, tt.session)
				assert.Equal(t, chatagent.ModePlan, sess.Mode)
			}
			if tt.msgAlt == "proceed" && tt.session != "" {
				sess := getTestChatSession(t, tt.session)
				assert.Equal(t, chatagent.ModeNormal, sess.Mode)
			}
		})
	}
}

func TestChatScope(t *testing.T) {
	tests := []struct {
		name string
		msg  protocol.MessageEventData
		want string
	}{
		{name: "thread id preferred", msg: protocol.MessageEventData{ThreadId: "T1", TopicId: "C1"}, want: "T1"},
		{name: "topic fallback", msg: protocol.MessageEventData{TopicId: "C1"}, want: "C1"},
		{name: "default fallback", msg: protocol.MessageEventData{}, want: "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatScope(tt.msg))
		})
	}
}

func TestChatCommandThreadID(t *testing.T) {
	msg := protocol.MessageEventData{ThreadId: "1700000000.000100"}
	tests := []struct {
		name   string
		msgAlt string
		want   string
	}{
		{name: "chat uses thread", msgAlt: "chat", want: "1700000000.000100"},
		{name: "end uses thread", msgAlt: "end", want: "1700000000.000100"},
		{name: "plan uses thread", msgAlt: "plan", want: "1700000000.000100"},
		{name: "proceed uses thread", msgAlt: "proceed", want: "1700000000.000100"},
		{name: "version has no thread", msgAlt: "version", want: ""},
		{name: "help has no thread", msgAlt: "help", want: ""},
		{name: "empty has no thread", msgAlt: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatCommandThreadID(msg, tt.msgAlt))
		})
	}
}

func TestThreadSessionIsolation(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, ctx types.Context, uid types.Uid)
	}{
		{
			name: "different threads get different sessions",
			run: func(t *testing.T, ctx types.Context, uid types.Uid) {
				_, s1 := manageChatSession(ctx, uid, "thread-a", "chat", "", nil)
				require.NotEmpty(t, s1)
				s2 := ensureThreadSession(ctx, uid, "thread-b", "", "hello")
				require.NotEmpty(t, s2)
				assert.NotEqual(t, s1, s2)
				assert.Equal(t, s1, loadThreadSessionID(ctx, uid, "thread-a"))
				assert.Equal(t, s2, loadThreadSessionID(ctx, uid, "thread-b"))
			},
		},
		{
			name: "same thread reuses session",
			run: func(t *testing.T, ctx types.Context, uid types.Uid) {
				_, s1 := manageChatSession(ctx, uid, "thread-a", "chat", "", nil)
				require.NotEmpty(t, s1)
				loaded := loadThreadSessionID(ctx, uid, "thread-a")
				assert.Equal(t, s1, loaded)
				again := ensureThreadSession(ctx, uid, "thread-a", loaded, "hello")
				assert.Equal(t, s1, again)
			},
		},
		{
			name: "enabled chat auto-creates session for new thread",
			run: func(t *testing.T, ctx types.Context, uid types.Uid) {
				_, s1 := manageChatSession(ctx, uid, "thread-a", "chat", "", nil)
				require.NotEmpty(t, s1)
				require.True(t, isChatEnabled(ctx, uid))
				s2 := ensureThreadSession(ctx, uid, "thread-b", "", "你是谁")
				require.NotEmpty(t, s2)
				assert.NotEqual(t, s1, s2)
			},
		},
		{
			name: "end closes all thread sessions",
			run: func(t *testing.T, ctx types.Context, uid types.Uid) {
				_, s1 := manageChatSession(ctx, uid, "thread-a", "chat", "", nil)
				require.NotEmpty(t, s1)
				s2 := ensureThreadSession(ctx, uid, "thread-b", "", "hello")
				require.NotEmpty(t, s2)

				payload, session := manageChatSession(ctx, uid, "thread-a", "end", s1, nil)
				assert.Equal(t, types.TextMsg{Text: "Chat ended"}, payload)
				assert.Empty(t, session)
				assert.False(t, isChatEnabled(ctx, uid))
				assert.Empty(t, loadThreadSessionID(ctx, uid, "thread-a"))
				assert.Empty(t, loadThreadSessionID(ctx, uid, "thread-b"))
				assert.Empty(t, ensureThreadSession(ctx, uid, "thread-c", "", "hello"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestCacheStore(t)
			setupSQLiteTestDB(t)
			ctx := types.Context{}
			ctx.SetContext(t.Context())
			tt.run(t, ctx, types.Uid("uid-iso"))
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
			name:   "help command builds MarkdownMsg with module rules",
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
				md, ok := got.(types.MarkdownMsg)
				require.True(t, ok)
				assert.Equal(t, "Help", md.Title)
				assert.Contains(t, md.Raw, "*"+modName+"*")
				assert.Contains(t, md.Raw, "`/test_cmd` — Test command")
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestFormatGroupedHelpMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		byModule map[string][]command.Rule
		want     string
	}{
		{
			name: "groups and sorts modules and commands",
			byModule: map[string][]command.Rule{
				"hub": {
					{Define: "version", Help: "Print version"},
					{Define: "deploy", Help: "deploy server"},
				},
				"example": {
					{Define: "event test", Help: "event example"},
				},
			},
			want: "*example*\n`/event test` — event example\n\n*hub*\n`/deploy` — deploy server\n`/version` — Print version",
		},
		{
			name:     "empty map returns empty string",
			byModule: map[string][]command.Rule{},
			want:     "",
		},
		{
			name: "modules with no rules are skipped",
			byModule: map[string][]command.Rule{
				"empty": {},
				"keep":  {{Define: "ping", Help: "Ping"}},
			},
			want: "*keep*\n`/ping` — Ping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatGroupedHelpMarkdown(tt.byModule))
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
