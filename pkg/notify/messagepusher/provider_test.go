package messagepusher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoSend(t *testing.T) {
	tests := []struct {
		name    string
		tokens  types.KV
		message notify.Message
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful send",
			tokens: types.KV{
				"user":    "testuser",
				"channel": "testchannel",
				"token":   "testtoken",
			},
			message: notify.Message{
				Title: "Test Title",
				Body:  "Test Body",
				Url:   "https://example.com",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/", r.URL.Path)
				assert.Equal(t, "testchannel", r.URL.Query().Get("channel"))
				assert.Equal(t, "testtoken", r.URL.Query().Get("token"))
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(Response{Message: "ok", Success: true})
			},
			wantErr: false,
		},
		{
			name: "api returns success false",
			tokens: types.KV{
				"user":    "testuser",
				"channel": "testchannel",
				"token":   "testtoken",
			},
			message: notify.Message{Title: "Test"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(Response{Message: "invalid token", Success: false})
			},
			wantErr: true,
		},
		{
			name: "server returns 500",
			tokens: types.KV{
				"user":    "testuser",
				"channel": "testchannel",
				"token":   "testtoken",
			},
			message: notify.Message{Title: "Test"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name:   "empty tokens",
			tokens: types.KV{},
			message: notify.Message{
				Title: "Test",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(Response{Message: "ok", Success: true})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := resty.New()
			err := doSend(tt.tokens, tt.message, client, srv.URL)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
