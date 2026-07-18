package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGrafana(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:3000", wantNil: false},
		{name: "with path", endpoint: "http://localhost:3000/grafana", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewGrafana(tt.endpoint, "tok")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestGrafana_Health(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       Health
		nilRecv    bool
		nilHTTP    bool
		wantErr    bool
	}{
		{
			name:       "ok",
			statusCode: http.StatusOK,
			body:       Health{Database: "ok", Version: "11.0.0"},
		},
		{
			name:       "server error",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{name: "nil receiver", nilRecv: true, wantErr: true},
		{name: "nil http client", nilHTTP: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var client *Grafana
			switch {
			case tt.nilRecv:
				client = nil
			case tt.nilHTTP:
				client = &Grafana{}
			default:
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/health", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					if tt.statusCode == http.StatusOK {
						_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
					}
				}))
				defer server.Close()
				client = NewGrafana(server.URL, "tok")
			}
			got, err := client.Health(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.body.Version, got.Version)
		})
	}
}

func TestGrafana_ListAndSearch(t *testing.T) {
	t.Parallel()
	t.Run("list datasources", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/datasources", r.URL.Path)
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			_ = sonic.ConfigDefault.NewEncoder(w).Encode([]Datasource{{ID: 1, Name: "Prometheus", Type: "prometheus"}})
		}))
		defer server.Close()
		got, err := NewGrafana(server.URL, "tok").ListDatasources(context.Background())
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
	t.Run("search dashboards", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/search", r.URL.Path)
			assert.Equal(t, "dash-db", r.URL.Query().Get("type"))
			assert.Equal(t, "flow", r.URL.Query().Get("query"))
			_ = sonic.ConfigDefault.NewEncoder(w).Encode([]DashboardHit{{UID: "abc", Title: "Flowbot"}})
		}))
		defer server.Close()
		got, err := NewGrafana(server.URL, "tok").SearchDashboards(context.Background(), "flow")
		require.NoError(t, err)
		assert.Equal(t, "abc", got[0].UID)
	})
	t.Run("search error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()
		_, err := NewGrafana(server.URL, "tok").SearchDashboards(context.Background(), "")
		assert.Error(t, err)
	})
}
