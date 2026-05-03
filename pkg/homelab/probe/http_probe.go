package probe

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

// HTTPProbe makes HTTP requests to discover API endpoints and determine
// authentication mechanisms on running containers.
type HTTPProbe struct {
	client  *http.Client
	timeout time.Duration
	auth    *AuthDetector
}

// NewHTTPProbe creates an HTTPProbe with the given timeout for each request.
func NewHTTPProbe(timeout time.Duration) *HTTPProbe {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &HTTPProbe{
		client:  client,
		timeout: timeout,
		auth:    &AuthDetector{},
	}
}

// EndpointProbeResult holds the outcome of a single endpoint probe attempt.
type EndpointProbeResult struct {
	BaseURL   string
	HealthURL string
	Auth      *homelab.AuthInfo
	Matches   []ProbeMatch
}

// ProbeEndpoint attempts to discover API information from a given base URL.
func (p *HTTPProbe) ProbeEndpoint(ctx context.Context, baseURL string) *EndpointProbeResult {
	if baseURL == "" {
		return nil
	}

	baseURL = strings.TrimRight(baseURL, "/")
	result := &EndpointProbeResult{BaseURL: baseURL}

	// Probe the base URL without auth to detect auth mechanism.
	authInfo := p.probeURL(ctx, baseURL)
	if authInfo != nil {
		result.Auth = authInfo
	}

	// Discover health endpoint from common paths.
	result.HealthURL = p.discoverHealth(ctx, baseURL)

	// Check for OIDC well-known discovery endpoint.
	if p.hasOIDCDiscovery(ctx, baseURL) {
		if authInfo == nil || authInfo.Type == homelab.AuthNone {
			result.Auth = &homelab.AuthInfo{
				Type:   homelab.AuthOIDC,
				Header: "Authorization",
				Prefix: "Bearer",
			}
		}
		authInfo = result.Auth
	}

	// Fingerprint matching for known services.
	if authInfo != nil {
		result.Matches = p.matchFingerprints(ctx, baseURL, authInfo)
	}

	return result
}

func (p *HTTPProbe) probeURL(ctx context.Context, rawURL string) *homelab.AuthInfo {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Flowbot-Homelab-Probe/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return p.auth.Detect(resp)
}

func (p *HTTPProbe) discoverHealth(ctx context.Context, baseURL string) string {
	healthPaths := []string{"/health", "/healthz", "/api/health", "/api/v1/health", "/ping", "/status"}
	for _, path := range healthPaths {
		healthURL, err := url.JoinPath(baseURL, path)
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Flowbot-Homelab-Probe/1.0")
		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return path
		}
	}
	return ""
}

func (p *HTTPProbe) matchFingerprints(ctx context.Context, baseURL string, authInfo *homelab.AuthInfo) []ProbeMatch {
	var matches []ProbeMatch
	for _, fp := range KnownServices {
		score := 0.0
		for _, pattern := range fp.Patterns {
			switch pattern.Field {
			case "path":
				if pattern.Key != "" {
					targetURL, err := url.JoinPath(baseURL, pattern.Key)
					if err != nil {
						continue
					}
					if p.pathReachable(ctx, targetURL) {
						score += 0.5
					}
				}
			}
		}
		if score > 0 {
			matches = append(matches, ProbeMatch{
				Capability: homelab.AppCapability{
					Capability: fp.Capability,
					Backend:    fp.Provider,
					Endpoint: &homelab.EndpointInfo{
						BaseURL: baseURL,
					},
					Auth: authInfo,
				},
				Confidence:  score,
				Fingerprint: fp.Provider,
			})
		}
	}
	return matches
}

// hasOIDCDiscovery probes the well-known OpenID Connect configuration endpoint
// to determine whether the service supports OIDC authentication.
func (p *HTTPProbe) hasOIDCDiscovery(ctx context.Context, baseURL string) bool {
	wellKnownURL, err := url.JoinPath(baseURL, "/.well-known/openid-configuration")
	if err != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Flowbot-Homelab-Probe/1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *HTTPProbe) pathReachable(ctx context.Context, rawURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Flowbot-Homelab-Probe/1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}
