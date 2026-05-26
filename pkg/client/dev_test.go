package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantTitle  string
		wantCPU    string
		wantErr    bool
		errContain string
	}{
		{
			name: "returns example data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"title":"Test","cpu":"2%","mem":"4GB","disk":"100GB"}}`))
			},
			wantTitle: "Test",
			wantCPU:   "2%",
			wantErr:   false,
		},
		{
			name: "handles missing fields returns empty strings",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			},
			wantTitle: "",
			wantCPU:   "",
			wantErr:   false,
		},
		{
			name: "handles partial data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"title":"OnlyTitle"}}`))
			},
			wantTitle: "OnlyTitle",
			wantCPU:   "",
			wantErr:   false,
		},
		{
			name: "api error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"service unavailable"}`))
			},
			wantErr:    true,
			errContain: "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Dev.Example(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantTitle, result.Title)
			assert.Equal(t, tt.wantCPU, result.CPU)
		})
	}
}
