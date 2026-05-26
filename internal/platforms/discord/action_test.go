package discord

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type mockChannelSender struct {
	msg  string
	err  error
	last string
}

func (m *mockChannelSender) ChannelMessageSend(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.last = content
	if m.err != nil {
		return nil, m.err
	}
	return &discordgo.Message{ID: "mock-msg-1", Content: content}, nil
}

func TestActionSendMessageWithMock(t *testing.T) {
	tests := []struct {
		name       string
		req        protocol.Request
		prepare    func(*mockChannelSender)
		wantStatus protocol.ResponseStatus
	}{
		{
			name: "sends text message successfully",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic": "C123",
					"message": protocol.Message{
						{Type: "text", Data: map[string]any{"text": "hello world"}},
					},
				},
			},
			prepare:    nil,
			wantStatus: protocol.Success,
		},
		{
			name: "sends url message successfully",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic": "D456",
					"message": protocol.Message{
						{Type: "url", Data: map[string]any{"url": "https://example.com"}},
					},
				},
			},
			prepare:    nil,
			wantStatus: protocol.Success,
		},
		{
			name: "empty message returns success without API call",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": protocol.Message{},
				},
			},
			prepare:    nil,
			wantStatus: protocol.Success,
		},
		{
			name: "nil message type returns error",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": nil,
				},
			},
			prepare:    nil,
			wantStatus: protocol.Failed,
		},
		{
			name: "wrong message type returns error",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": "not a protocol.Message",
				},
			},
			prepare:    nil,
			wantStatus: protocol.Failed,
		},
		{
			name: "send failure returns error",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic": "C999",
					"message": protocol.Message{
						{Type: "text", Data: map[string]any{"text": "hello"}},
					},
				},
			},
			prepare: func(m *mockChannelSender) {
				m.err = errors.New("network error")
			},
			wantStatus: protocol.Failed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockChannelSender{}
			if tt.prepare != nil {
				tt.prepare(mock)
			}
			action := &Action{session: mock}
			resp := action.SendMessage(tt.req)
			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, resp.Status)
			}
		})
	}
}

func TestActionSendMessageMultiSegment(t *testing.T) {
	mock := &mockChannelSender{}
	action := &Action{session: mock}
	resp := action.SendMessage(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: map[string]any{
			"topic": "C123",
			"message": protocol.Message{
				{Type: "text", Data: map[string]any{"text": "line1"}},
				{Type: "text", Data: map[string]any{"text": "line2"}},
			},
		},
	})
	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s", resp.Status)
	}
	if mock.last != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got %q", mock.last)
	}
}

func TestActionSendMessageIgnoresNonTextUrls(t *testing.T) {
	mock := &mockChannelSender{}
	action := &Action{session: mock}
	resp := action.SendMessage(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: map[string]any{
			"topic": "C123",
			"message": protocol.Message{
				{Type: "text", Data: map[string]any{"text": "hello"}},
				{Type: "markdown", Data: map[string]any{"text": "ignored"}},
				{Type: "image", Data: map[string]any{"file_id": "F123"}},
			},
		},
	})
	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s", resp.Status)
	}
	if mock.last != "hello" {
		t.Errorf("expected 'hello', got %q", mock.last)
	}
}

func TestUnsupportedActionsDiscord(t *testing.T) {
	action := &Action{}

	tests := []struct {
		name string
		call func() protocol.Response
	}{
		{name: "GetLatestEvents", call: func() protocol.Response { return action.GetLatestEvents(protocol.Request{}) }},
		{name: "GetSupportedActions", call: func() protocol.Response { return action.GetSupportedActions(protocol.Request{}) }},
		{name: "GetStatus", call: func() protocol.Response { return action.GetStatus(protocol.Request{}) }},
		{name: "GetVersion", call: func() protocol.Response { return action.GetVersion(protocol.Request{}) }},
		{name: "GetUserInfo", call: func() protocol.Response { return action.GetUserInfo(protocol.Request{}) }},
		{name: "CreateChannel", call: func() protocol.Response { return action.CreateChannel(protocol.Request{}) }},
		{name: "GetChannelInfo", call: func() protocol.Response { return action.GetChannelInfo(protocol.Request{}) }},
		{name: "GetChannelList", call: func() protocol.Response { return action.GetChannelList(protocol.Request{}) }},
		{name: "RegisterChannels", call: func() protocol.Response { return action.RegisterChannels(protocol.Request{}) }},
		{name: "RegisterSlashCommands", call: func() protocol.Response { return action.RegisterSlashCommands(protocol.Request{}) }},
		{name: "UpdateMessage", call: func() protocol.Response { return action.UpdateMessage(protocol.Request{}) }},
		{name: "DeleteMessage", call: func() protocol.Response { return action.DeleteMessage(protocol.Request{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.call()
			if resp.Status != protocol.Failed {
				t.Errorf("expected Failed, got %s", resp.Status)
			}
			if resp.RetCode != "10002" {
				t.Errorf("expected retcode 10002, got %s", resp.RetCode)
			}
		})
	}
}
