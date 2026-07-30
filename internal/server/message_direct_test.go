package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

// helpDispatchTestModule implements module.Handler for direct-message help dispatch tests.
type helpDispatchTestModule struct {
	module.Base
	ready  bool
	define string
	help   string
}

func (h *helpDispatchTestModule) IsReady() bool              { return h.ready }
func (*helpDispatchTestModule) Init(_ json.RawMessage) error { return nil }

func (h *helpDispatchTestModule) Rules() []any {
	return []any{[]command.Rule{{
		Define: h.define,
		Help:   h.help,
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "command-ok"}
		},
	}}}
}

func (h *helpDispatchTestModule) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return module.RunCommand(h.Rules()[0].([]command.Rule), ctx, content)
}

// resolveDirectModulePayload mirrors dispatchDirectMessage payload resolution for tests.
func resolveDirectModulePayload(sessionID, msgAlt string, payload types.MsgPayload, ctx types.Context) types.MsgPayload {
	if sessionID == "" && payload == nil {
		payload = dispatchToModules(ctx, msgAlt)
	}
	return payload
}

func TestBuildDirectMessageContextSetsTopicAndPlatform(t *testing.T) {
	tests := []struct {
		name         string
		data         protocol.MessageEventData
		wantTopic    string
		wantPlatform string
	}{
		{
			name: "sets topic and platform on module context",
			data: protocol.MessageEventData{
				Self:       protocol.Self{Platform: "slack", UserId: "U01DMQDTV5W"},
				UserId:     "U01DMQDTV5W",
				TopicId:    "D06EN8RGU6S",
				TopicType:  "im",
				MessageId:  "msg-1",
				AltMessage: "hub health",
			},
			wantTopic:    "existing-channel",
			wantPlatform: "slack",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSQLiteTestDB(t)
			platformID := seedTestPlatform(t, "slack")
			user := seedTestUser(t, "user-flag")
			seedTestPlatformUser(t, platformID, user.ID, "U01DMQDTV5W", "", "")
			channelID := seedTestChannel(t, "existing-channel")
			seedTestPlatformChannel(t, platformID, channelID, "D06EN8RGU6S")

			dmCtx, err := buildDirectMessageContext(t.Context(), "evt-1", tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTopic, dmCtx.topic)
			assert.Equal(t, tt.wantTopic, dmCtx.ctx.Topic)
			assert.Equal(t, tt.wantPlatform, dmCtx.ctx.Platform)
		})
	}
}

func TestIsDuplicateDirectMessage(t *testing.T) {
	tests := []struct {
		name          string
		messageID     string
		seedMessage   bool
		wantDuplicate bool
	}{
		{
			name:          "empty message id skips lookup",
			messageID:     "",
			wantDuplicate: false,
		},
		{
			name:          "missing stored message is not duplicate",
			messageID:     "msg-1",
			wantDuplicate: false,
		},
		{
			name:          "existing stored message is duplicate",
			messageID:     "msg-1",
			seedMessage:   true,
			wantDuplicate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSQLiteTestDB(t)
			platformID := seedTestPlatform(t, "slack")
			if tt.seedMessage {
				now := time.Now()
				require.NoError(t, store.MessageStoreFromDB().CreateMessage(t.Context(), gen.Message{
					Flag:          types.Id(),
					PlatformID:    platformID,
					PlatformMsgID: tt.messageID,
					Topic:         "direct",
					Session:       "sess-dm",
					Role:          types.User,
					State:         int(schema.MessageCreated),
					CreatedAt:     now,
					UpdatedAt:     now,
				}))
			}

			dmCtx := directMessageContext{
				ctx:        types.Context{},
				platformID: platformID,
				msg: protocol.MessageEventData{
					MessageId: tt.messageID,
				},
			}
			dmCtx.ctx.SetContext(t.Context())

			got := isDuplicateDirectMessage(dmCtx)
			assert.Equal(t, tt.wantDuplicate, got)
		})
	}
}

func TestPersistDirectUserMessage(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		wantCreated bool
	}{
		{
			name:        "active session persists user message",
			sessionID:   "sess-active",
			wantCreated: true,
		},
		{
			name:        "empty session is skipped by caller",
			sessionID:   "",
			wantCreated: false,
		},
		{
			name:        "closed session id still persists when explicitly provided",
			sessionID:   "sess-closed",
			wantCreated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSQLiteTestDB(t)
			platformID := seedTestPlatform(t, "slack")

			dmCtx := directMessageContext{
				ctx:        types.Context{},
				platformID: platformID,
				topic:      "topic-flag",
			}
			dmCtx.ctx.SetContext(t.Context())
			msg := protocol.MessageEventData{
				MessageId:  "msg-1",
				AltMessage: "hello",
			}

			var persisted bool
			if tt.sessionID != "" {
				persisted = persistDirectUserMessage(dmCtx, tt.sessionID, msg)
			}

			if tt.wantCreated {
				require.True(t, persisted)
				stored, err := store.MessageStoreFromDB().GetMessageByPlatform(t.Context(), platformID, "msg-1")
				require.NoError(t, err)
				require.NotNil(t, stored)
				assert.Equal(t, tt.sessionID, stored.Session)
				assert.Equal(t, "msg-1", stored.PlatformMsgID)
				assert.Equal(t, types.User, stored.Role)
				assert.Equal(t, int(schema.MessageCreated), stored.State)
			} else {
				assert.False(t, persisted)
			}
		})
	}
}

func TestResolveDirectModulePayload(t *testing.T) {
	tests := []struct {
		name        string
		msgAlt      string
		sessionID   string
		wantNil     bool
		wantHelpKey string
		wantText    string
	}{
		{
			name:        "help keeps aggregated commands from all modules",
			msgAlt:      "help",
			sessionID:   "",
			wantHelpKey: "[help-mod-a] /alpha-cmd",
		},
		{
			name:      "non-help command dispatches when payload is nil",
			msgAlt:    "alpha-cmd",
			sessionID: "",
			wantText:  "command-ok",
		},
		{
			name:      "active chat session skips module dispatch",
			msgAlt:    "alpha-cmd",
			sessionID: "sess-1",
			wantNil:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module.Register("help-mod-a", &helpDispatchTestModule{
				ready:  true,
				define: "alpha-cmd",
				help:   "alpha help",
			})
			module.Register("help-mod-b", &helpDispatchTestModule{
				ready:  true,
				define: "beta-cmd",
				help:   "beta help",
			})
			t.Cleanup(func() {
				module.Unregister("help-mod-a")
				module.Unregister("help-mod-b")
			})

			ctx := types.Context{}
			payload := buildHelpMessage(tt.msgAlt, nil)
			got := resolveDirectModulePayload(tt.sessionID, tt.msgAlt, payload, ctx)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			if tt.wantText != "" {
				text, ok := got.(types.TextMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantText, text.Text)
				return
			}
			info, ok := got.(types.InfoMsg)
			require.True(t, ok)
			assert.Contains(t, info.Model, tt.wantHelpKey)
			if tt.msgAlt == "help" {
				assert.Contains(t, info.Model, "[help-mod-b] /beta-cmd")
			}
		})
	}
}
