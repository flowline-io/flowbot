package tailchat

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestAdapterEventConvertGroupVsDirect(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name          string
		groupID       string
		wantDetail    string
		wantTopicType string
	}{
		{
			name:          "with GroupID produces MessageGroupEvent",
			groupID:       "G123",
			wantDetail:    protocol.MessageGroupEvent,
			wantTopicType: "group",
		},
		{
			name:          "empty GroupID produces MessageDirectEvent",
			groupID:       "",
			wantDetail:    protocol.MessageDirectEvent,
			wantTopicType: "dm",
		},
		{
			name:          "specific GroupID produces MessageGroupEvent",
			groupID:       "G999",
			wantDetail:    protocol.MessageGroupEvent,
			wantTopicType: "group",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &Payload{
				ID:     "evt-test",
				UserID: "bot-id",
				Type:   "message",
				Payload: PayloadData{
					GroupID:             tt.groupID,
					ConverseID:          "C123",
					MessageID:           "msg-test",
					MessageAuthor:       "U999",
					MessagePlainContent: "test",
				},
			}
			result := adapter.EventConvert(payload)
			if result.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %s, got %s", tt.wantDetail, result.DetailType)
			}
			data, ok := result.Data.(protocol.MessageEventData)
			if !ok {
				t.Fatal("expected MessageEventData")
			}
			if data.TopicType != tt.wantTopicType {
				t.Errorf("expected topicType %s, got %s", tt.wantTopicType, data.TopicType)
			}
		})
	}
}

func TestAdapterEventConvertValidPayload(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name    string
		payload *Payload
		check   func(*testing.T, protocol.Event)
	}{
		{
			name: "group event with full data",
			payload: &Payload{
				ID: "evt-1", UserID: "self-bot", Type: "message",
				Payload: PayloadData{GroupID: "G123", ConverseID: "C456", MessageID: "msg-001", MessageAuthor: "U789", MessagePlainContent: "hello group"},
			},
			check: func(t *testing.T, evt protocol.Event) {
				if evt.Id == "" || evt.DetailType != protocol.MessageGroupEvent || evt.Type != protocol.MessageEventType {
					t.Errorf("unexpected event: %+v", evt)
				}
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok {
					t.Fatal("expected MessageEventData")
				}
				if data.Self.Platform != ID {
					t.Errorf("expected platform %s, got %s", ID, data.Self.Platform)
				}
				if data.UserId != "U789" || data.TopicId != "C456" || data.AltMessage != "hello group" || data.MessageId != "msg-001" {
					t.Errorf("unexpected data: %+v", data)
				}
			},
		},
		{
			name: "group event with empty content",
			payload: &Payload{
				ID: "evt-2", UserID: "self-bot", Type: "message",
				Payload: PayloadData{GroupID: "G456", ConverseID: "C789", MessageID: "msg-002", MessageAuthor: "U333", MessagePlainContent: ""},
			},
			check: func(t *testing.T, evt protocol.Event) {
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok {
					t.Fatal("expected MessageEventData")
				}
				if data.AltMessage != "" {
					t.Errorf("expected empty altMessage, got %s", data.AltMessage)
				}
			},
		},
		{
			name: "self message returns empty",
			payload: &Payload{
				ID: "evt-3", UserID: "self-bot", Type: "message",
				Payload: PayloadData{GroupID: "G123", ConverseID: "C456", MessageID: "msg-003", MessageAuthor: "self-bot", MessagePlainContent: "self"},
			},
			check: func(t *testing.T, evt protocol.Event) {
				if evt.DetailType != "" {
					t.Errorf("expected empty DetailType for self, got %s", evt.DetailType)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.EventConvert(tt.payload)
			tt.check(t, result)
		})
	}
}

func TestAdapterEventConvertEdgeCasesTailchat(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name string
		data any
	}{
		{name: "nil input", data: nil},
		{name: "non-Payload type", data: "not a payload"},
		{name: "int input", data: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.data)
			if evt.DetailType != "" || evt.Id != "" {
				t.Error("expected empty event")
			}
		})
	}
}

func TestAdapterMessageConvertTailchat(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name  string
		input any
	}{
		{name: "nil input", input: nil},
		{name: "string input", input: "plain text"},
		{name: "int input", input: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.MessageConvert(tt.input)
			if len(result) != 1 || result[0].Type != "text" {
				t.Errorf("unexpected result: %+v", result)
			}
		})
	}
}
