package slack

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestAdapterEventConvertHello(t *testing.T) {
	tests := []struct {
		name string
		data socketmode.Event
	}{
		{name: "Hello event", data: socketmode.Event{Type: socketmode.EventTypeHello}},
		{name: "Connecting event returns empty", data: socketmode.Event{Type: socketmode.EventTypeConnecting}},
		{name: "ConnectionError event returns empty", data: socketmode.Event{Type: socketmode.EventTypeConnectionError}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{}
			evt := adapter.EventConvert(tt.data)
			if tt.name == "Hello event" {
				if evt.Id == "" || evt.Type != protocol.MetaEventType || evt.DetailType != protocol.MetaConnectEvent {
					t.Errorf("unexpected event: %+v", evt)
				}
			} else {
				if evt.DetailType != "" {
					t.Errorf("expected empty DetailType, got %s", evt.DetailType)
				}
			}
		})
	}
}

func TestAdapterEventConvertEventsAPI(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name       string
		data       socketmode.Event
		wantDetail string
		wantUserId string
		decode     func(*testing.T, protocol.Event)
	}{
		{
			name: "IM channel produces MessageDirectEvent",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{ChannelType: "im", Channel: "C123", User: "U456", Text: "hello", ClientMsgID: "msg-001"},
				}},
			},
			wantDetail: protocol.MessageDirectEvent,
			wantUserId: "U456",
		},
		{
			name: "channel produces MessageGroupEvent",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{ChannelType: "channel", Channel: "G789", User: "U111", Text: "group", ClientMsgID: "msg-002"},
				}},
			},
			wantDetail: protocol.MessageGroupEvent,
			wantUserId: "U111",
		},
		{
			name: "bot message returns empty",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{BotID: "B123", Channel: "C123", User: "B456", Text: "bot"},
				}},
			},
			wantDetail: "",
			wantUserId: "",
		},
		{
			name: "non-message inner event returns empty",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "app_mention",
					Data: &slackevents.MessageEvent{Channel: "C123", User: "U456"},
				}},
			},
			wantDetail: "",
			wantUserId: "",
		},
		{
			name: "message subtype is ignored",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{
						SubType:     "message_changed",
						ChannelType: "im",
						Channel:     "D01DMRLE0HW",
					},
				}},
			},
			wantDetail: "",
			wantUserId: "",
		},
		{
			name: "incomplete im message without user is ignored",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{
						ChannelType: "im",
						Channel:     "D01DMRLE0HW",
					},
				}},
			},
			wantDetail: "",
			wantUserId: "",
		},
		{
			name: "im message without client_msg_id falls back to ts",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{
						ChannelType: "im",
						Channel:     "D01DMRLE0HW",
						User:        "U456",
						Text:        "hello",
						TimeStamp:   "1781250417.829078",
					},
				}},
			},
			wantDetail: protocol.MessageDirectEvent,
			wantUserId: "U456",
			decode: func(t *testing.T, evt protocol.Event) {
				t.Helper()
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok {
					t.Fatal("expected MessageEventData")
				}
				if data.MessageId != "1781250417.829078" {
					t.Errorf("expected message id from ts, got %q", data.MessageId)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.data)
			if evt.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %q, got %q", tt.wantDetail, evt.DetailType)
			}
			if tt.wantUserId != "" {
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok {
					t.Fatal("expected MessageEventData")
				}
				if data.UserId != tt.wantUserId {
					t.Errorf("expected userId %s, got %s", tt.wantUserId, data.UserId)
				}
				if data.Self.Platform != ID {
					t.Errorf("expected platform %s, got %s", ID, data.Self.Platform)
				}
			}
			if tt.decode != nil {
				tt.decode(t, evt)
			}
		})
	}
}

func TestAdapterEventConvertSlashCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "slash command", command: "/example"},
		{name: "empty command", command: ""},
		{name: "complex command", command: "/test --verbose"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{}
			evt := adapter.EventConvert(socketmode.Event{
				Type: socketmode.EventTypeSlashCommand,
				Data: slack.SlashCommand{Command: tt.command},
			})
			if evt.Id == "" {
				t.Error("expected non-empty Id")
			}
			if evt.DetailType != protocol.MessageCommandEvent {
				t.Errorf("expected MessageCommandEvent, got %s", evt.DetailType)
			}
			cmdData, ok := evt.Data.(protocol.CommandEventData)
			if !ok {
				t.Fatal("expected CommandEventData")
			}
			if cmdData.Command != tt.command {
				t.Errorf("expected command %q, got %q", tt.command, cmdData.Command)
			}
		})
	}
}

func TestAdapterEventConvertInteractive(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name string
		data socketmode.Event
		want string
	}{
		{
			name: "BlockActions",
			data: socketmode.Event{
				Type: socketmode.EventTypeInteractive,
				Data: slack.InteractionCallback{
					Type: slack.InteractionTypeBlockActions,
					ActionCallback: slack.ActionCallbacks{
						BlockActions: []*slack.BlockAction{{ActionID: "btn", Value: "confirmed"}},
					},
					Channel:   slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C456"}}},
					User:      slack.User{ID: "U789"},
					MessageTs: "1700000000.000100",
				},
			},
			want: protocol.MessageDirectEvent,
		},
		{
			name: "ViewSubmission",
			data: socketmode.Event{
				Type: socketmode.EventTypeInteractive,
				Data: slack.InteractionCallback{
					Type: slack.InteractionTypeViewSubmission,
					View: slack.View{
						CallbackID: "cb1",
						State:      &slack.ViewState{Values: map[string]map[string]slack.BlockAction{}},
					},
					User: slack.User{ID: "U222"},
				},
			},
			want: protocol.MessageDirectEvent,
		},
		{
			name: "Shortcut returns empty",
			data: socketmode.Event{
				Type: socketmode.EventTypeInteractive,
				Data: slack.InteractionCallback{Type: slack.InteractionTypeShortcut, CallbackID: "sc1"},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.data)
			if evt.DetailType != tt.want {
				t.Errorf("expected DetailType %q, got %q", tt.want, evt.DetailType)
			}
		})
	}
}

func TestAdapterEventConvertEdgeCases(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name string
		data any
		want string
	}{
		{name: "nil input", data: nil, want: ""},
		{name: "non-socketmode.Event", data: "not an event", want: ""},
		{
			name: "events API without ThreadTimeStamp",
			data: socketmode.Event{
				Type: socketmode.EventTypeEventsAPI,
				Data: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
					Type: "message",
					Data: &slackevents.MessageEvent{ChannelType: "im", Channel: "C999", User: "U333", Text: "hello"},
				}},
			},
			want: protocol.MessageDirectEvent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.data)
			if evt.DetailType != tt.want {
				t.Errorf("expected DetailType %q, got %q", tt.want, evt.DetailType)
			}
		})
	}
}

func TestConvertEventsAPIEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      slackevents.EventsAPIEvent
		wantDetail string
	}{
		{
			name: "IM returns direct event",
			input: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: "message",
				Data: &slackevents.MessageEvent{ChannelType: "im", Channel: "D123", User: "U001", Text: "hello", ClientMsgID: "c1"},
			}},
			wantDetail: protocol.MessageDirectEvent,
		},
		{
			name: "non-message type returns empty",
			input: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: "reaction_added", Data: (*slackevents.MessageEvent)(nil),
			}},
			wantDetail: "",
		},
		{
			name: "channel returns group event",
			input: slackevents.EventsAPIEvent{InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: "message",
				Data: &slackevents.MessageEvent{ChannelType: "channel", Channel: "C456", User: "U002", Text: "group", ClientMsgID: "c2"},
			}},
			wantDetail: protocol.MessageGroupEvent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertEventsAPIEvent(tt.input)
			if result.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %q, got %q", tt.wantDetail, result.DetailType)
			}
		})
	}
}

func TestConvertInteractiveEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      *slack.InteractionCallback
		wantDetail string
		check      func(*testing.T, protocol.Event)
	}{
		{
			name: "BlockActions",
			input: &slack.InteractionCallback{
				Type: slack.InteractionTypeBlockActions,
				ActionCallback: slack.ActionCallbacks{
					BlockActions: []*slack.BlockAction{{ActionID: "act", Value: "val"}},
				},
				Channel:   slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}},
				User:      slack.User{ID: "U1"},
				MessageTs: "ts-1",
			},
			wantDetail: protocol.MessageDirectEvent,
			check: func(t *testing.T, evt protocol.Event) {
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok || data.AltMessage != "val" {
					t.Error("expected AltMessage=val")
				}
			},
		},
		{
			name: "ViewSubmission",
			input: &slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					CallbackID: "cb1",
					State:      &slack.ViewState{Values: map[string]map[string]slack.BlockAction{}},
				},
				User: slack.User{ID: "U2"},
			},
			wantDetail: protocol.MessageDirectEvent,
			check: func(t *testing.T, evt protocol.Event) {
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok || data.Option != "form_submit" {
					t.Error("expected Option=form_submit")
				}
			},
		},
		{
			name: "Shortcut returns empty",
			input: &slack.InteractionCallback{
				Type: slack.InteractionTypeShortcut, CallbackID: "sc1",
			},
			wantDetail: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := convertInteractiveEvent(tt.input)
			if evt.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %q, got %q", tt.wantDetail, evt.DetailType)
			}
			if tt.check != nil {
				tt.check(t, evt)
			}
		})
	}
}

func TestSetAndGetThreadContext(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		thread  string
	}{
		{name: "store and retrieve", channel: "C123", thread: "1700000000.000100"},
		{name: "empty thread ts", channel: "C456", thread: ""},
		{name: "overwrite existing", channel: "C789", thread: "1700000000.000200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "overwrite existing" {
				setThreadContext(tt.channel, "old")
			}
			setThreadContext(tt.channel, tt.thread)
			if got := getThreadContext(tt.channel); got != tt.thread {
				t.Errorf("expected %q, got %q", tt.thread, got)
			}
		})
	}
}

func TestAdapterMessageConvertSlack(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name  string
		input any
	}{
		{name: "nil input", input: nil},
		{name: "string input", input: "plain text"},
		{name: "struct input", input: struct{ Text string }{Text: "test"}},
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
