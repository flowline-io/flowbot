package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSSE(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTypes []string
		wantErr   bool
	}{
		{
			name: "delta and done",
			input: "data: {\"type\":\"delta\",\"text\":\"hi\"}\n\n" +
				"data: {\"type\":\"done\",\"text\":\"hi\"}\n\n",
			wantTypes: []string{"delta", "done"},
		},
		{
			name:      "confirm resolved",
			input:     "data: {\"type\":\"confirm_resolved\",\"id\":\"c1\",\"approved\":false,\"reason\":\"timeout\"}\n\n",
			wantTypes: []string{"confirm_resolved"},
		},
		{
			name:    "invalid json",
			input:   "data: {bad}\n\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var types []string
			err := readSSE(strings.NewReader(tt.input), func(ev ChatStreamEvent) error {
				types = append(types, ev.Type)
				return nil
			})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTypes, types)
		})
	}
}

func TestSendMessageSSEHonorsContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, cc *ChatAgentClient)
	}{
		{
			name: "canceled context aborts blocked handler",
			run: func(t *testing.T, cc *ChatAgentClient) {
				t.Helper()
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				err := cc.SendMessageSSE(ctx, "sess-1", "hi", func(ChatStreamEvent) error {
					<-ctx.Done()
					return ctx.Err()
				})
				assert.Error(t, err)
			},
		},
		{
			name: "request context stops before headers",
			run: func(t *testing.T, cc *ChatAgentClient) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				err := cc.SendMessageSSE(ctx, "sess-1", "hi", nil)
				assert.Error(t, err)
			},
		},
		{
			name: "unreachable host returns quickly",
			run: func(t *testing.T, _ *ChatAgentClient) {
				t.Helper()
				cl := NewClient("http://127.0.0.1:1", "token")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := cl.ChatAgent.SendMessageSSE(ctx, "sess-1", "hi", nil)
				assert.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				<-r.Context().Done()
			}))
			t.Cleanup(srv.Close)

			cl := NewClient(srv.URL, "token")
			tt.run(t, cl.ChatAgent)
		})
	}
}

func TestExportSession(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		status     int
		body       string
		wantErr    bool
		wantCount  int
		wantSessID string
	}{
		{
			name:       "returns full export",
			sessionID:  "sess-1",
			status:     http.StatusOK,
			body:       `{"session_id":"sess-1","entry_count":2,"entries":[{"id":"e1"},{"id":"e2"}]}`,
			wantCount:  2,
			wantSessID: "sess-1",
		},
		{
			name:      "server error",
			sessionID: "sess-1",
			status:    http.StatusInternalServerError,
			body:      `{"error":"boom"}`,
			wantErr:   true,
		},
		{
			name:       "empty session",
			sessionID:  "sess-2",
			status:     http.StatusOK,
			body:       `{"session_id":"sess-2","entry_count":0,"entries":[]}`,
			wantCount:  0,
			wantSessID: "sess-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/chatagent/sessions/"+tt.sessionID+"/export", r.URL.Path)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)

			cl := NewClient(srv.URL, "token")
			export, err := cl.ChatAgent.ExportSession(context.Background(), tt.sessionID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSessID, export.SessionID)
			assert.Equal(t, tt.wantCount, export.EntryCount)
		})
	}
}
