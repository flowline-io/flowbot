package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestOnInstructPushEventHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name:    "empty payload returns nil",
			payload: []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "valid instruct push event returns nil",
			payload: []byte(`{"id":"evt-1","type":"instruct"}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON returns nil (handler ignores errors)",
			payload: []byte(`not-json`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := message.NewMessage("test-uuid", tt.payload)
			err := onInstructPushEventHandler(msg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOnMessageSendEventHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name: "empty platform and topic returns nil (ignored)",
			payload: func() []byte {
				b, _ := sonic.Marshal(types.Message{Platform: "", Topic: ""})
				return b
			}(),
			wantErr: false,
		},
		{
			name: "empty topic returns nil (ignored)",
			payload: func() []byte {
				b, _ := sonic.Marshal(types.Message{Platform: "discord", Topic: ""})
				return b
			}(),
			wantErr: false,
		},
		{
			name:    "invalid JSON returns error",
			payload: []byte(`not-json`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := message.NewMessage("test-uuid", tt.payload)
			err := onMessageSendEventHandler(msg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOnPlatformMessageEventHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name:    "invalid JSON returns error",
			payload: []byte(`not-json`),
			wantErr: true,
		},
		{
			name: "valid payload with data containing unknown platform returns error",
			payload: func() []byte {
				raw := struct {
					Id         string             `json:"id"`
					Time       int64              `json:"time"`
					Type       protocol.EventType `json:"type"`
					DetailType string             `json:"detail_type"`
					Data       json.RawMessage    `json:"data"`
				}{
					Id: "evt-test", Time: 100, Type: protocol.MessageEventType,
					DetailType: "message.direct",
				}
				raw.Data, _ = sonic.Marshal(protocol.MessageEventData{
					Self:   protocol.Self{Platform: "discord"},
					UserId: "u-test",
				})
				b, _ := sonic.Marshal(raw)
				return b
			}(),
			wantErr: true,
		},
		{
			name: "valid payload with missing self platform returns error",
			payload: func() []byte {
				raw := struct {
					Id         string             `json:"id"`
					Time       int64              `json:"time"`
					Type       protocol.EventType `json:"type"`
					DetailType string             `json:"detail_type"`
					Data       json.RawMessage    `json:"data"`
				}{
					Id: "evt-test", Time: 100, Type: protocol.MessageEventType,
					DetailType: "message.group",
				}
				raw.Data, _ = sonic.Marshal(protocol.MessageEventData{
					Self:   protocol.Self{Platform: ""},
					UserId: "u-test",
				})
				b, _ := sonic.Marshal(raw)
				return b
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := message.NewMessage("test-uuid", tt.payload)
			err := onPlatformMessageEventHandler(msg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleEvents_RequiresValidLifecycle(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "handleEvents panics with nil lc (requires fx lifecycle)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := watermill.NewStdLogger(false, false)
			pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, logger)
			defer pubSub.Close()

			router, err := message.NewRouter(message.RouterConfig{CloseTimeout: time.Second}, logger)
			require.NoError(t, err)

			// Close router before checking panic to avoid 30s handler drain timeout.
			defer func() {
				_ = router.Close()
			}()

			defer func() {
				r := recover()
				assert.NotNil(t, r, "expected panic with nil fx.Lifecycle")
			}()

			_ = handleEvents(nil, router, pubSub)
		})
	}
}

func TestOnPlatformMessageEventHandler_EmptyPlatform(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "empty platform in event data returns error from GetCaller"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := message.NewMessage("id-1", []byte(`{"id":"evt-1","time":123,"type":"message","data":{"self":{"platform":""},"user_id":""}}`))
			err := onPlatformMessageEventHandler(msg)
			assert.Error(t, err)
		})
	}
}

func TestOnInstructPushEventHandler_ContextPropagation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "instruct push handler accepts context"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := message.NewMessage("ctx-id", []byte(`{}`))
			type ctxKey struct{}
			msg.SetContext(context.WithValue(msg.Context(), ctxKey{}, "val"))
			err := onInstructPushEventHandler(msg)
			assert.NoError(t, err)
		})
	}
}
