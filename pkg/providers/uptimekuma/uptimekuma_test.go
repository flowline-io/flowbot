package uptimekuma

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUptimeKuma(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint returns nil", endpoint: "", wantNil: true},
		{name: "configured endpoint", endpoint: "https://uptime.example.com", wantNil: false},
		{name: "localhost", endpoint: "http://127.0.0.1:8002", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewUptimeKuma(tt.endpoint, "tok")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestUptimeKuma_Metrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantErr    bool
		wantFamily string
	}{
		{
			name: "successful metrics retrieval",
			body: `# HELP uptimekuma_monitor_status Status of monitored services (1=up, 0=down)
# TYPE uptimekuma_monitor_status gauge
uptimekuma_monitor_status{monitor_name="Google",monitor_type="http"} 1
# HELP uptimekuma_monitor_response_time Response time in ms
# TYPE uptimekuma_monitor_response_time gauge
uptimekuma_monitor_response_time{monitor_name="Google"} 25
`,
			statusCode: http.StatusOK,
			wantFamily: "uptimekuma_monitor_status",
		},
		{
			name:       "invalid prometheus response",
			body:       "not valid prometheus metrics",
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "empty metrics response",
			body:       "",
			statusCode: http.StatusOK,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/metrics", r.URL.Path)
				w.Header().Set("Content-Type", "text/plain; version=0.0.4")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewUptimeKuma(server.URL, "my-api-token")
			result, err := client.Metrics(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantFamily != "" {
				_, ok := result[tt.wantFamily]
				assert.True(t, ok)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestUptimeKuma_Health(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "healthy", statusCode: http.StatusOK},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/metrics", r.URL.Path)
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_, _ = w.Write([]byte("# TYPE up gauge\nup 1\n"))
				}
			}))
			defer server.Close()
			err := NewUptimeKuma(server.URL, "tok").Health(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
