package slack

import (
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
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message: notify.Message{
				Title: "Test Title",
				Body:  "Test Body",
				Url:   "https://example.com",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "server returns 400",
			tokens: types.KV{
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message: notify.Message{Title: "Test"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
		{
			name: "server returns 500",
			tokens: types.KV{
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message: notify.Message{Title: "Test"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "empty tokens",
			tokens: types.KV{},
			message: notify.Message{
				Title: "Test",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
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
