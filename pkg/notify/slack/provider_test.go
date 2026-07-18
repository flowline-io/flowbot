package slack

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoSend(t *testing.T) {
	tests := []struct {
		name         string
		tokens       types.KV
		message      notify.Message
		status       int
		respBody     string
		wantErr      bool
		wantErrSub   string
		wantBlocks   int
		wantUsername string
	}{
		{
			name: "successful send includes url block",
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
			status:     http.StatusOK,
			wantErr:    false,
			wantBlocks: 2,
		},
		{
			name: "omits empty url block so slack accepts payload",
			tokens: types.KV{
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message: notify.Message{
				Title: "Test Notification",
				Body:  "Connectivity test from Flowbot",
			},
			status:     http.StatusOK,
			wantErr:    false,
			wantBlocks: 1,
		},
		{
			name: "sets username when botname present",
			tokens: types.KV{
				"botname": "flowbot",
				"tokenA":  "T00000000",
				"tokenB":  "B00000000",
				"tokenC":  "C000000000000000",
			},
			message:      notify.Message{Title: "Test", Body: "Body"},
			status:       http.StatusOK,
			wantErr:      false,
			wantBlocks:   1,
			wantUsername: "flowbot",
		},
		{
			name: "server returns 400 with body in error",
			tokens: types.KV{
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message:    notify.Message{Title: "Test", Body: "Body"},
			status:     http.StatusBadRequest,
			respBody:   "invalid_blocks",
			wantErr:    true,
			wantErrSub: "invalid_blocks",
			wantBlocks: 1,
		},
		{
			name: "server returns 500",
			tokens: types.KV{
				"tokenA": "T00000000",
				"tokenB": "B00000000",
				"tokenC": "C000000000000000",
			},
			message:    notify.Message{Title: "Test", Body: "Body"},
			status:     http.StatusInternalServerError,
			wantErr:    true,
			wantBlocks: 1,
		},
		{
			name:   "empty tokens",
			tokens: types.KV{},
			message: notify.Message{
				Title: "Test",
				Body:  "Body",
			},
			status:     http.StatusOK,
			wantErr:    false,
			wantBlocks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu        sync.Mutex
				gotBody   []byte
				gotMethod string
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				mu.Lock()
				gotMethod = r.Method
				gotBody = body
				mu.Unlock()
				w.WriteHeader(tt.status)
				if tt.respBody != "" {
					_, _ = w.Write([]byte(tt.respBody))
				}
			}))
			defer srv.Close()

			client := resty.New()
			err := doSend(tt.tokens, tt.message, client, srv.URL)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrSub != "" {
					assert.Contains(t, err.Error(), tt.wantErrSub)
				}
			} else {
				require.NoError(t, err)
			}

			mu.Lock()
			method := gotMethod
			raw := append([]byte(nil), gotBody...)
			mu.Unlock()

			assert.Equal(t, http.MethodPost, method)
			var payload map[string]any
			require.NoError(t, sonic.Unmarshal(raw, &payload))
			blocks, ok := payload["blocks"].([]any)
			require.True(t, ok)
			assert.Len(t, blocks, tt.wantBlocks)
			if tt.wantUsername != "" {
				assert.Equal(t, tt.wantUsername, payload["username"])
			}
		})
	}
}
