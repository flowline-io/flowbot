package wakapi

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWakapi(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:3000", wantNil: false},
		{name: "with path", endpoint: "http://localhost:3000/api", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewWakapi(tt.endpoint, "key")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestWakapi_GetSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("my-key"))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/summary", r.URL.Path)
				assert.Equal(t, "today", r.URL.Query().Get("interval"))
				assert.Equal(t, wantAuth, r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(Summary{Total: 120})
				}
			}))
			defer server.Close()
			got, err := NewWakapi(server.URL, "my-key").GetSummary(context.Background(), "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(120), got.Total)
		})
	}
}

func TestWakapi_ListProjects(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/compat/wakatime/v1/users/current/projects", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(ProjectsResponse{
				Data: []Project{{ID: "1", Name: "flowbot"}},
			})
		}))
		defer server.Close()
		got, err := NewWakapi(server.URL, "key").ListProjects(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "flowbot", got[0].Name)
	})
	t.Run("error status", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()
		_, err := NewWakapi(server.URL, "key").ListProjects(context.Background())
		assert.Error(t, err)
	})
}
