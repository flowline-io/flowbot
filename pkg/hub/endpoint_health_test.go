package hub

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestEndpointHealthChecker_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		healthURL  string
		handler    http.HandlerFunc
		wantStatus HealthStatus
		wantErr    bool
	}{
		{
			name:       "empty URL is healthy",
			healthURL:  "",
			wantStatus: HealthHealthy,
		},
		{
			name: "2xx is healthy",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatus: HealthHealthy,
		},
		{
			name: "non-2xx is unhealthy",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantStatus: HealthUnhealthy,
		},
		{
			name:       "unreachable URL is unhealthy",
			healthURL:  "http://127.0.0.1:1/",
			wantStatus: HealthUnhealthy,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checker := NewEndpointHealthChecker(2 * time.Second)
			healthURL := tt.healthURL
			if tt.handler != nil {
				srv := httptest.NewServer(tt.handler)
				t.Cleanup(srv.Close)
				healthURL = srv.URL
			}
			status, err := checker.Check(t.Context(), healthURL)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestEndpointHealthChecker_CheckCapabilities(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failSrv.Close)

	tests := []struct {
		name    string
		apps    []homelab.App
		hubCaps []Descriptor
		want    []CapabilityHealth
	}{
		{
			name: "empty health stays healthy without probe",
			apps: []homelab.App{
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{
							Capability: "bookmark",
							Endpoint:   &homelab.EndpointInfo{BaseURL: okSrv.URL, Health: ""},
						},
					},
				},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthHealthy},
			},
		},
		{
			name: "nil endpoint stays healthy",
			apps: []homelab.App{
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{Capability: "bookmark"},
					},
				},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthHealthy},
			},
		},
		{
			name: "non-2xx is unhealthy",
			apps: []homelab.App{
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{
							Capability: "bookmark",
							Endpoint:   &homelab.EndpointInfo{BaseURL: failSrv.URL, Health: "/health"},
						},
					},
				},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthUnhealthy},
			},
		},
		{
			name: "registered hub capability is skipped",
			apps: []homelab.App{
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{
							Capability: string(CapKarakeep),
							Endpoint:   &homelab.EndpointInfo{BaseURL: okSrv.URL, Health: "/health"},
						},
						{
							Capability: "bookmark",
							Endpoint:   &homelab.EndpointInfo{BaseURL: okSrv.URL, Health: "/health"},
						},
					},
				},
			},
			hubCaps: []Descriptor{
				{Type: CapKarakeep, App: "alpha", Healthy: true, Instance: "ok"},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthHealthy},
			},
		},
		{
			name: "invalid base URL is unhealthy",
			apps: []homelab.App{
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{
							Capability: "bookmark",
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://%", Health: "/health"},
						},
					},
				},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthUnhealthy},
			},
		},
		{
			name: "preserves apps by name then capability order",
			apps: []homelab.App{
				{
					Name: "bravo",
					Capabilities: []homelab.AppCapability{
						{
							Capability: "rss",
							Endpoint:   &homelab.EndpointInfo{BaseURL: okSrv.URL, Health: "/health"},
						},
					},
				},
				{
					Name: "alpha",
					Capabilities: []homelab.AppCapability{
						{
							Capability: "bookmark",
							Endpoint:   &homelab.EndpointInfo{BaseURL: failSrv.URL, Health: "/health"},
						},
						{
							Capability: "note",
							Endpoint:   &homelab.EndpointInfo{BaseURL: okSrv.URL, Health: "/health"},
						},
					},
				},
			},
			want: []CapabilityHealth{
				{Type: "bookmark", App: "alpha", Status: HealthUnhealthy},
				{Type: "note", App: "alpha", Status: HealthHealthy},
				{Type: "rss", App: "bravo", Status: HealthHealthy},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := homelab.DefaultRegistry.List()
			homelab.DefaultRegistry.Replace(tt.apps)
			t.Cleanup(func() { homelab.DefaultRegistry.Replace(old) })

			registry := NewRegistry()
			for _, d := range tt.hubCaps {
				require.NoError(t, registry.Register(d))
			}

			checker := NewEndpointHealthChecker(2 * time.Second)
			got := checker.CheckCapabilities(t.Context(), registry)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i].Type, got[i].Type, "index %d type", i)
				assert.Equal(t, tt.want[i].App, got[i].App, "index %d app", i)
				assert.Equal(t, tt.want[i].Status, got[i].Status, "index %d status", i)
				if tt.want[i].Status == HealthUnhealthy && tt.name == "invalid base URL is unhealthy" {
					assert.NotEmpty(t, got[i].Description)
				}
			}
		})
	}
}

func TestEndpointHealthChecker_CheckCapabilities_Concurrent(t *testing.T) {
	const delay = 200 * time.Millisecond
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slow.Close)

	old := homelab.DefaultRegistry.List()
	homelab.DefaultRegistry.Replace([]homelab.App{
		{
			Name: "alpha",
			Capabilities: []homelab.AppCapability{
				{
					Capability: "bookmark",
					Endpoint:   &homelab.EndpointInfo{BaseURL: slow.URL, Health: "/a"},
				},
				{
					Capability: "note",
					Endpoint:   &homelab.EndpointInfo{BaseURL: slow.URL, Health: "/b"},
				},
			},
		},
	})
	t.Cleanup(func() { homelab.DefaultRegistry.Replace(old) })

	checker := NewEndpointHealthChecker(2 * time.Second)
	start := time.Now()
	got := checker.CheckCapabilities(t.Context(), NewRegistry())
	elapsed := time.Since(start)

	require.Len(t, got, 2)
	assert.Equal(t, HealthHealthy, got[0].Status)
	assert.Equal(t, HealthHealthy, got[1].Status)
	// Serial would be ~400ms; concurrent should finish near one delay.
	assert.Less(t, elapsed, 350*time.Millisecond, "want concurrent probes, elapsed=%s", elapsed)
}

func TestChecker_Check_ReusesSharedEndpointChecker(t *testing.T) {
	old := homelab.DefaultRegistry.List()
	homelab.DefaultRegistry.Replace(nil)
	t.Cleanup(func() { homelab.DefaultRegistry.Replace(old) })

	shared := defaultEndpointHealthChecker
	require.NotNil(t, shared)

	registry := NewRegistry()
	require.NoError(t, registry.Register(Descriptor{
		Type: CapExample, App: "example", Healthy: true, Instance: "ok",
	}))
	checker := NewChecker(registry)

	first := checker.Check(t.Context())
	assert.Same(t, shared, defaultEndpointHealthChecker)
	second := checker.Check(t.Context())
	assert.Same(t, shared, defaultEndpointHealthChecker)

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.Equal(t, HealthHealthy, first.Status)
	assert.Equal(t, HealthHealthy, second.Status)
}
