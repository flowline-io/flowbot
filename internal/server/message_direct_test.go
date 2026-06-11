package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type messageDirectStore struct {
	testStoreAdapter
	createdMessage *gen.Message
	createErr      error
}

func (s *messageDirectStore) CreateMessage(_ context.Context, message gen.Message) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createdMessage = &message
	return nil
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
