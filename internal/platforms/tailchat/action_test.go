package tailchat

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func newTestClient(t *testing.T, srv *httptest.Server) *client {
	t.Helper()
	c := resty.New()
	c.SetBaseURL(srv.URL)
	return &client{c: c, clientId: "test-app", clientSecret: "test-secret"}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = sonic.ConfigDefault.NewEncoder(w).Encode(data)
}

func TestClientAuth(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantErr   bool
		wantToken string
	}{
		{
			name: "successful auth",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/openapi/bot/login" {
					http.Error(w, "not found", 404)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(TokenResponse{Data: TokenData{Jwt: "jwt-token-123"}})
			},
			wantErr:   false,
			wantToken: "jwt-token-123",
		},
		{
			name: "auth returns non-200",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr:   true,
			wantToken: "",
		},
		{
			name: "auth returns unexpected JSON structure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, map[string]string{"foo": "bar"})
			},
			wantErr:   false,
			wantToken: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			cl := newTestClient(t, srv)
			err := cl.auth()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if cl.accessToken != tt.wantToken {
				t.Errorf("expected token %q, got %q", tt.wantToken, cl.accessToken)
			}
		})
	}
}

func TestClientSendMessage(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		data    SendMessageData
		wantErr bool
	}{
		{
			name: "successful send",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/chat/message/sendMessage" {
					http.Error(w, "not found", 404)
					return
				}
				if r.Header.Get("X-Token") != "token-abc" {
					http.Error(w, "unauthorized", 401)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
			data: SendMessageData{
				ConverseId: "C123",
				Content:    "hello",
				Plain:      "hello",
			},
			wantErr: false,
		},
		{
			name: "send returns non-200",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			data:    SendMessageData{ConverseId: "C456", Content: "test"},
			wantErr: true,
		},
		{
			name: "send with network error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				// Close connection to simulate error - just send 404
				http.Error(w, "not found", 404)
			},
			data:    SendMessageData{ConverseId: "C789", Content: "msg"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			cl := newTestClient(t, srv)
			cl.accessToken = "token-abc"

			err := cl.sendMessage(tt.data)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestActionSendMessageTailchat(t *testing.T) {
	tests := []struct {
		name       string
		req        protocol.Request
		wantStatus protocol.ResponseStatus
		serverErr  bool
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
			wantStatus: protocol.Success,
		},
		{
			name: "empty message returns success without send",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": protocol.Message{},
				},
			},
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
			wantStatus: protocol.Failed,
		},
		{
			name: "server error returns failed response",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic": "C999",
					"message": protocol.Message{
						{Type: "text", Data: map[string]any{"text": "hello"}},
					},
				},
			},
			wantStatus: protocol.Failed,
			serverErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				switch r.URL.Path {
				case "/api/openapi/bot/login":
					writeJSON(w, http.StatusOK, TokenResponse{Data: TokenData{Jwt: "test-jwt"}})
				case "/api/chat/message/sendMessage":
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer srv.Close()

			cl := newTestClient(t, srv)
			cl.accessToken = "test-jwt"
			action := &Action{client: cl}

			resp := action.SendMessage(tt.req)
			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, resp.Status)
			}
		})
	}
}

func TestTailchatSendMessageMultiSegment(t *testing.T) {
	tests := []struct {
		name    string
		message protocol.Message
		want    string
	}{
		{
			name: "two text segments concatenated",
			message: protocol.Message{
				{Type: "text", Data: map[string]any{"text": "line1"}},
				{Type: "text", Data: map[string]any{"text": "line2"}},
			},
			want: "line1\nline2",
		},
		{
			name: "text and url combo",
			message: protocol.Message{
				{Type: "text", Data: map[string]any{"text": "Check: "}},
				{Type: "url", Data: map[string]any{"url": "https://a.com"}},
			},
			want: "Check: \nhttps://a.com",
		},
		{
			name: "non-text segments ignored",
			message: protocol.Message{
				{Type: "text", Data: map[string]any{"text": "hello"}},
				{Type: "markdown", Data: map[string]any{"text": "ignored"}},
				{Type: "image", Data: map[string]any{"file_id": "F123"}},
			},
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/chat/message/sendMessage" {
					var body SendMessageData
					sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
					if body.Content != tt.want {
						t.Errorf("expected content %q, got %q", tt.want, body.Content)
					}
				}
				writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
			}))
			defer srv.Close()

			cl := newTestClient(t, srv)
			cl.accessToken = "test-token"
			action := &Action{client: cl}

			resp := action.SendMessage(protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": tt.message,
				},
			})
			if resp.Status != protocol.Success {
				t.Errorf("expected success, got %s", resp.Status)
			}
		})
	}
}

func TestTailchatAuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/openapi/bot/login":
			writeJSON(w, http.StatusOK, TokenResponse{Data: TokenData{Jwt: "test-jwt"}})
		case "/api/chat/message/sendMessage":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cl := newTestClient(t, srv)
	action := &Action{client: cl}

	resp := action.SendMessage(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: map[string]any{
			"topic":   "C123",
			"message": protocol.Message{{Type: "text", Data: map[string]any{"text": "hello"}}},
		},
	})
	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	if cl.accessToken != "test-jwt" {
		t.Errorf("expected token 'test-jwt' after auto-auth, got %q", cl.accessToken)
	}
}

func TestUnsupportedActionsTailchat(t *testing.T) {
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
