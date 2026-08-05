package hub

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

const (
	defaultEndpointHealthTimeout = 5 * time.Second
	endpointHealthConcurrency    = 8
)

// defaultEndpointHealthChecker is shared across health checks so callers reuse
// one http.Client instead of allocating per Check().
var defaultEndpointHealthChecker = NewEndpointHealthChecker(defaultEndpointHealthTimeout)

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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
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

type endpointProbeJob struct {
	idx       int
	healthURL string
}

// CheckCapabilities probes all discovered endpoint health URLs across
// all homelab apps and builds CapabilityHealth entries. Capabilities that are
// already registered in the hub registry are skipped to avoid duplicates.
// Probes run concurrently (bounded) while preserving apps×capabilities order.
func (c *EndpointHealthChecker) CheckCapabilities(ctx context.Context, registry *Registry) []CapabilityHealth {
	apps := homelab.DefaultRegistry.List()
	results := make([]CapabilityHealth, 0)
	jobs := make([]endpointProbeJob, 0)

	for _, app := range apps {
		for _, cap := range app.Capabilities {
			if registry != nil {
				if _, ok := registry.Get(CapabilityType(cap.Capability)); ok {
					continue
				}
			}
			ch := CapabilityHealth{
				Type:   CapabilityType(cap.Capability),
				App:    app.Name,
				Status: HealthHealthy,
			}
			if cap.Endpoint != nil && cap.Endpoint.Health != "" {
				healthURL, joinErr := url.JoinPath(cap.Endpoint.BaseURL, cap.Endpoint.Health)
				if joinErr != nil {
					ch.Status = HealthUnhealthy
					ch.Description = joinErr.Error()
					results = append(results, ch)
					continue
				}
				jobs = append(jobs, endpointProbeJob{idx: len(results), healthURL: healthURL})
			}
			results = append(results, ch)
		}
	}

	if len(jobs) == 0 {
		return results
	}

	sem := make(chan struct{}, endpointHealthConcurrency)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			status, err := c.Check(ctx, job.healthURL)
			if err != nil {
				results[job.idx].Status = HealthUnhealthy
				results[job.idx].Description = err.Error()
				return
			}
			results[job.idx].Status = status
		})
	}
	wg.Wait()
	return results
}
