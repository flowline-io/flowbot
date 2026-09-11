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
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
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

// baseResponse holds the captured base URL response for auth detection.
type baseResponse struct {
	status  int
	headers http.Header
}

// ProbeEndpoint attempts to discover API information from a given base URL.
func (p *HTTPProbe) ProbeEndpoint(ctx context.Context, baseURL string) *EndpointProbeResult {
	if baseURL == "" {
		return nil
	}

	baseURL = strings.TrimRight(baseURL, "/")
	result := &EndpointProbeResult{BaseURL: baseURL}

	br := p.fetchBase(ctx, baseURL)
	if br != nil {
		result.Auth = p.auth.Detect(makeSyntheticResponse(br))
	}

	result.HealthURL = p.discoverHealth(ctx, baseURL)

	if p.hasOIDCDiscovery(ctx, baseURL) {
		if result.Auth == nil || result.Auth.Type == homelab.AuthNone {
			result.Auth = &homelab.AuthInfo{
				Type:   homelab.AuthOIDC,
				Header: "Authorization",
				Prefix: "Bearer",
			}
		}
	}

	if result.Auth != nil {
		result.Matches = p.matchFingerprints(ctx, baseURL, result.Auth)
	}

	return result
}

// fetchBase retrieves the base URL status and headers for auth detection.
func (p *HTTPProbe) fetchBase(ctx context.Context, rawURL string) *baseResponse {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Flowbot-Homelab-Probe/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return &baseResponse{
		status:  resp.StatusCode,
		headers: resp.Header,
	}
}

// makeSyntheticResponse builds a minimal http.Response for auth detection
// from captured base response data.
func makeSyntheticResponse(br *baseResponse) *http.Response {
	return &http.Response{
		StatusCode: br.status,
		Header:     br.headers,
	}
}

func (p *HTTPProbe) discoverHealth(ctx context.Context, baseURL string) string {
	healthPaths := []string{"/health", "/healthz", "/api/health", "/api/v1/health", "/ping", "/status"}
	for _, path := range healthPaths {
		healthURL, err := url.JoinPath(baseURL, path)
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
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
		for _, path := range fp.Paths {
			if path == "" {
				continue
			}
			targetURL, err := url.JoinPath(baseURL, path)
			if err != nil {
				continue
			}
			if p.pathReachable(ctx, targetURL) {
				score += 0.5
			}
		}
		if score > 0 {
			matches = append(matches, ProbeMatch{
				Capability: homelab.AppCapability{
					Capability: fp.Capability,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, http.NoBody)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
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
