package hub

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

// EndpointHealthChecker probes HTTP endpoints discovered on homelab apps to
// determine their health status.
type EndpointHealthChecker struct {
	client  *http.Client
	timeout time.Duration
}

// NewEndpointHealthChecker creates a health checker with the given HTTP
// request timeout.
func NewEndpointHealthChecker(timeout time.Duration) *EndpointHealthChecker {
	return &EndpointHealthChecker{
		client:  &http.Client{Timeout: timeout},
		timeout: timeout,
	}
}

// Check probes a single health URL and returns whether the endpoint is
// healthy.
func (c *EndpointHealthChecker) Check(ctx context.Context, healthURL string) (HealthStatus, error) {
	if healthURL == "" {
		return HealthHealthy, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return HealthUnhealthy, err
	}
	req.Header.Set("User-Agent", "Flowbot-HealthCheck/1.0")
	resp, err := c.client.Do(req)
	if err != nil {
		return HealthUnhealthy, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return HealthHealthy, nil
	}
	return HealthUnhealthy, nil
}

// CheckCapabilities probes all discovered endpoint health URLs across
// all homelab apps and builds CapabilityHealth entries. Capabilities that are
// already registered in the hub registry are skipped to avoid duplicates.
func (c *EndpointHealthChecker) CheckCapabilities(ctx context.Context, registry *Registry) []CapabilityHealth {
	apps := homelab.DefaultRegistry.List()
	var results []CapabilityHealth
	for _, app := range apps {
		for _, cap := range app.Capabilities {
			// Skip if this capability is already registered in the hub.
			if registry != nil {
				if _, ok := registry.Get(CapabilityType(cap.Capability)); ok {
					continue
				}
			}
			ch := CapabilityHealth{
				Type:    CapabilityType(cap.Capability),
				Backend: cap.Backend,
				App:     app.Name,
				Status:  HealthHealthy,
			}
			if cap.Endpoint != nil && cap.Endpoint.Health != "" {
				healthURL, joinErr := url.JoinPath(cap.Endpoint.BaseURL, cap.Endpoint.Health)
				if joinErr != nil {
					ch.Status = HealthUnhealthy
					ch.Description = joinErr.Error()
					results = append(results, ch)
					continue
				}
				status, err := c.Check(ctx, healthURL)
				if err != nil {
					ch.Status = HealthUnhealthy
					ch.Description = err.Error()
				} else {
					ch.Status = status
				}
			}
			results = append(results, ch)
		}
	}
	return results
}
