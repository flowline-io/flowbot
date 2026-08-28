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

func TestWakapi_Health(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("app=1\ndb=1\n"))
	}))
	defer server.Close()
	got, err := NewWakapi(server.URL, "").Health(context.Background())
	require.NoError(t, err)
	assert.True(t, got.AppOK)
	assert.True(t, got.DBOK)
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
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(Summary{
						Projects: []SummaryItem{{Key: "flowbot", Total: 120}},
					})
				}
			}))
			defer server.Close()
			got, err := NewWakapi(server.URL, "my-key").GetSummary(context.Background(), SummaryParams{})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(120), got.TotalSeconds())
		})
	}
}

func TestWakapi_GetStats(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/compat/wakatime/v1/users/current/stats/week", r.URL.Path)
		assert.Equal(t, "flowbot", r.URL.Query().Get("project"))
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(StatsResponse{
			Data: Stats{
				TotalSeconds:       3600,
				HumanReadableTotal: "1 hr",
				Range:              "week",
			},
		})
	}))
	defer server.Close()
	got, err := NewWakapi(server.URL, "key").GetStats(context.Background(), StatsParams{
		Range:   "week",
		Project: "flowbot",
	})
	require.NoError(t, err)
	assert.InDelta(t, float64(3600), got.TotalSeconds, 0.001)
	assert.Equal(t, "1 hr", got.HumanReadableTotal)
}

func TestWakapi_GetAllTime(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/compat/wakatime/v1/users/current/all_time_since_today", r.URL.Path)
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(AllTimeResponse{
			Data: AllTime{TotalSeconds: 86400, Text: "1 day", IsUpToDate: true},
		})
	}))
	defer server.Close()
	got, err := NewWakapi(server.URL, "key").GetAllTime(context.Background(), "")
	require.NoError(t, err)
	assert.InDelta(t, float64(86400), got.TotalSeconds, 0.001)
	assert.True(t, got.IsUpToDate)
}

func TestWakapi_ListHeartbeats(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/compat/wakatime/v1/users/current/heartbeats", r.URL.Path)
		assert.Equal(t, "2026-08-28", r.URL.Query().Get("date"))
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(HeartbeatsResult{
			Data: []Heartbeat{{Project: "flowbot", Language: "Go"}},
		})
	}))
	defer server.Close()
	got, err := NewWakapi(server.URL, "key").ListHeartbeats(context.Background(), HeartbeatsParams{Date: "2026-08-28"})
	require.NoError(t, err)
	require.Len(t, got.Data, 1)
	assert.Equal(t, "flowbot", got.Data[0].Project)
}

func TestWakapi_ListHeartbeatsRequiresDate(t *testing.T) {
	t.Parallel()
	_, err := NewWakapi("http://localhost", "key").ListHeartbeats(context.Background(), HeartbeatsParams{})
	assert.Error(t, err)
}

func TestWakapi_ListProjects(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/compat/wakatime/v1/users/current/projects", r.URL.Path)
			assert.Equal(t, "flow", r.URL.Query().Get("q"))
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(ProjectsResponse{
				Data: []Project{{ID: "1", Name: "flowbot"}},
			})
		}))
		defer server.Close()
		got, err := NewWakapi(server.URL, "key").ListProjects(context.Background(), "flow")
		require.NoError(t, err)
		assert.Equal(t, "flowbot", got[0].Name)
	})
	t.Run("error status", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()
		_, err := NewWakapi(server.URL, "key").ListProjects(context.Background(), "")
		assert.Error(t, err)
	})
}

func TestParseHealthStatus(t *testing.T) {
	t.Parallel()
	got := parseHealthStatus("app=1\ndb=0\n")
	assert.True(t, got.AppOK)
	assert.False(t, got.DBOK)
}
