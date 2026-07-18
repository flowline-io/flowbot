package traefik

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTraefik(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:8080", wantNil: false},
		{name: "with trailing slash", endpoint: "http://localhost:8080/", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewTraefik(tt.endpoint, "u", "p")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestTraefik_ListRouters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       []Router
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			body:       []Router{{Name: "web@docker", Rule: "Host(`example.com`)", Status: "enabled"}},
			wantLen:    1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       []Router{},
			wantLen:    0,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/http/routers", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
				}
			}))
			defer server.Close()

			client := NewTraefik(server.URL, "", "")
			result, err := client.ListRouters(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestTraefik_OverviewAndServices(t *testing.T) {
	t.Parallel()
	t.Run("overview success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/overview", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(Overview{
				HTTP: &ProtocolStats{Routers: map[string]int{"total": 3}},
			})
		}))
		defer server.Close()
		client := NewTraefik(server.URL, "admin", "secret")
		got, err := client.Overview(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 3, got.HTTP.Routers["total"])
	})
	t.Run("list services success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/http/services", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode([]Service{{Name: "svc@docker", Type: "loadbalancer", Status: "enabled"}})
		}))
		defer server.Close()
		client := NewTraefik(server.URL, "", "")
		got, err := client.ListServices(context.Background())
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
	t.Run("list services error status", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		client := NewTraefik(server.URL, "", "")
		_, err := client.ListServices(context.Background())
		assert.Error(t, err)
	})
}
