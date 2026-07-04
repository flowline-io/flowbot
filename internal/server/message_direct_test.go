package server

import (
	"context"
	"encoding/json"
	"testing"

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

type messageDirectStore struct {
	testStoreAdapter
	createdMessage      *gen.Message
	createErr           error
	getByPlatformCalled bool
	getByPlatformErr    error
}

func (s *messageDirectStore) GetMessageByPlatform(_ context.Context, _ int64, platformMsgID string) (*gen.Message, error) {
	s.getByPlatformCalled = true
	if s.getByPlatformErr != nil {
		return nil, s.getByPlatformErr
	}
	if platformMsgID == "" {
		return nil, types.ErrNotFound
	}
	return nil, types.ErrNotFound
}

func (s *messageDirectStore) CreateMessage(_ context.Context, message gen.Message) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createdMessage = &message
	return nil
}

func TestBuildDirectMessageContextSetsTopicAndPlatform(t *testing.T) {
	tests := []struct {
		name          string
		data          protocol.MessageEventData
		wantTopic     string
		wantPlatform  string
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
			storeStub := &directMessageContextStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
				platformUser: &gen.PlatformUser{
					ID:     11,
					UserID: 42,
					Flag:   "U01DMQDTV5W",
				},
				user: &gen.User{ID: 42, Flag: "user-flag"},
				platformChannel: &gen.PlatformChannel{
					ID:        12,
					ChannelID: 88,
					Flag:      "D06EN8RGU6S",
				},
				channel: &gen.Channel{ID: 88, Flag: "existing-channel"},
			}
			orig := store.Database
			store.Database = storeStub
			t.Cleanup(func() { store.Database = orig })

			dmCtx, err := buildDirectMessageContext(t.Context(), "evt-1", tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTopic, dmCtx.topic)
			assert.Equal(t, tt.wantTopic, dmCtx.ctx.Topic)
			assert.Equal(t, tt.wantPlatform, dmCtx.ctx.Platform)
		})
	}
}

type directMessageContextStore struct {
	testStoreAdapter
	platform        *gen.Platform
	platformUser    *gen.PlatformUser
	user            *gen.User
	platformChannel *gen.PlatformChannel
	channel         *gen.Channel
}

func (s *directMessageContextStore) GetPlatformByName(_ context.Context, name string) (*gen.Platform, error) {
	if s.platform == nil || s.platform.Name != name {
		return nil, types.ErrNotFound
	}
	return s.platform, nil
}

func (s *directMessageContextStore) GetPlatformUserByFlag(_ context.Context, flag string) (*gen.PlatformUser, error) {
	if s.platformUser == nil || s.platformUser.Flag != flag {
		return nil, types.ErrNotFound
	}
	return s.platformUser, nil
}

func (s *directMessageContextStore) GetUserById(_ context.Context, id int64) (*gen.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, types.ErrNotFound
	}
	return s.user, nil
}

func (s *directMessageContextStore) GetPlatformChannelByFlag(_ context.Context, flag string) (*gen.PlatformChannel, error) {
	if s.platformChannel == nil || s.platformChannel.Flag != flag {
		return nil, types.ErrNotFound
	}
	return s.platformChannel, nil
}

func (s *directMessageContextStore) GetChannel(_ context.Context, id int64) (*gen.Channel, error) {
	if s.channel == nil || s.channel.ID != id {
		return nil, types.ErrNotFound
	}
	return s.channel, nil
}

func TestIsDuplicateDirectMessage(t *testing.T) {
	tests := []struct {
		name             string
		messageID        string
		wantDuplicate    bool
		wantLookup       bool
		getByPlatformErr error
	}{
		{
			name:          "empty message id skips lookup",
			messageID:     "",
			wantDuplicate: false,
			wantLookup:    false,
		},
		{
			name:          "missing stored message is not duplicate",
			messageID:     "msg-1",
			wantDuplicate: false,
			wantLookup:    true,
		},
		{
			name:             "lookup error is treated as duplicate guard",
			messageID:        "msg-1",
			wantDuplicate:    true,
			wantLookup:       true,
			getByPlatformErr: assert.AnError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeStub := &messageDirectStore{getByPlatformErr: tt.getByPlatformErr}
			orig := store.Database
			store.Database = storeStub
			t.Cleanup(func() { store.Database = orig })

			dmCtx := directMessageContext{
				ctx:        types.Context{},
				platformID: 7,
				msg: protocol.MessageEventData{
					MessageId: tt.messageID,
				},
			}
			dmCtx.ctx.SetContext(t.Context())

			got := isDuplicateDirectMessage(dmCtx)
			assert.Equal(t, tt.wantDuplicate, got)
			assert.Equal(t, tt.wantLookup, storeStub.getByPlatformCalled)
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
			storeStub := &messageDirectStore{}
			orig := store.Database
			store.Database = storeStub
			t.Cleanup(func() { store.Database = orig })

			dmCtx := directMessageContext{
				ctx:        types.Context{},
				platformID: 7,
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
				require.NotNil(t, storeStub.createdMessage)
				assert.Equal(t, tt.sessionID, storeStub.createdMessage.Session)
				assert.Equal(t, "msg-1", storeStub.createdMessage.PlatformMsgID)
				assert.Equal(t, types.User, storeStub.createdMessage.Role)
				assert.Equal(t, int(schema.MessageCreated), storeStub.createdMessage.State)
			} else {
				assert.False(t, persisted)
				assert.Nil(t, storeStub.createdMessage)
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
