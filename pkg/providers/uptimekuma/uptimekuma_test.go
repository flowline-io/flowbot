package uptimekuma

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUptimeKuma_Metrics(t *testing.T) {
	prometheusMetrics := `# HELP uptimekuma_monitor_status Status of monitored services (1=up, 0=down)
# TYPE uptimekuma_monitor_status gauge
uptimekuma_monitor_status{monitor_name="Google",monitor_type="http"} 1
uptimekuma_monitor_status{monitor_name="GitHub",monitor_type="http"} 1
uptimekuma_monitor_status{monitor_name="Test Service",monitor_type="ping"} 0
# HELP uptimekuma_monitor_response_time Response time in ms
# TYPE uptimekuma_monitor_response_time gauge
uptimekuma_monitor_response_time{monitor_name="Google"} 25
uptimekuma_monitor_response_time{monitor_name="GitHub"} 45
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/metrics", r.URL.Path)
		assert.Equal(t, "Basic Om15LWFwaS10b2tlbg==", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(prometheusMetrics))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewUptimeKuma(server.URL, "my-api-token")
	result, err := client.Metrics()

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that we got the expected metric families
	statusMetric, ok := result["uptimekuma_monitor_status"]
	require.True(t, ok)
	assert.Equal(t, "Status of monitored services (1=up, 0=down)", *statusMetric.Help)

	responseTimeMetric, ok := result["uptimekuma_monitor_response_time"]
	require.True(t, ok)
	assert.Equal(t, "Response time in ms", *responseTimeMetric.Help)
}

func TestUptimeKuma_Metrics_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("not valid prometheus metrics"))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewUptimeKuma(server.URL, "my-token")
	_, err := client.Metrics()

	assert.Error(t, err)
}

func TestUptimeKuma_Metrics_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(""))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewUptimeKuma(server.URL, "my-token")
	result, err := client.Metrics()

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNewUptimeKuma(t *testing.T) {
	client := NewUptimeKuma("https://uptime.example.com", "my-token")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}
