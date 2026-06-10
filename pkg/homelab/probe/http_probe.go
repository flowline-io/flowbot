package probe

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
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

// baseResponse holds the captured base URL response data for reuse
// in both auth detection and fingerprint matching.
type baseResponse struct {
	status  int
	headers http.Header
	body    []byte
}

// ProbeEndpoint attempts to discover API information from a given base URL.
func (p *HTTPProbe) ProbeEndpoint(ctx context.Context, baseURL string) *EndpointProbeResult {
	if baseURL == "" {
		return nil
	}

	baseURL = strings.TrimRight(baseURL, "/")
	result := &EndpointProbeResult{BaseURL: baseURL}

	// Probe the base URL without auth to detect auth mechanism and capture
	// response data for fingerprint matching.
	br := p.fetchBase(ctx, baseURL)
	if br != nil {
		result.Auth = p.auth.Detect(makeSyntheticResponse(br))
	}

	// Discover health endpoint from common paths.
	result.HealthURL = p.discoverHealth(ctx, baseURL)

	// Check for OIDC well-known discovery endpoint.
	if p.hasOIDCDiscovery(ctx, baseURL) {
		if result.Auth == nil || result.Auth.Type == homelab.AuthNone {
			result.Auth = &homelab.AuthInfo{
				Type:   homelab.AuthOIDC,
				Header: "Authorization",
				Prefix: "Bearer",
			}
		}
	}

	// Fingerprint matching for known services.
	if result.Auth != nil && br != nil {
		result.Matches = p.matchFingerprints(ctx, baseURL, br, result.Auth)
	}

	return result
}

// fetchBase retrieves the base URL response and captures headers and body
// for use in both auth detection and fingerprint matching.
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return nil
	}
	return &baseResponse{
		status:  resp.StatusCode,
		headers: resp.Header,
		body:    body,
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

func (p *HTTPProbe) matchFingerprints(ctx context.Context, baseURL string, br *baseResponse, authInfo *homelab.AuthInfo) []ProbeMatch {
	var matches []ProbeMatch
	for _, fp := range KnownServices {
		score := 0.0
		for _, pattern := range fp.Patterns {
			switch pattern.Field {
			case "header":
				if matchHeader(br.headers, pattern.Key, pattern.Value) {
					score += 0.5
				}
			case "title":
				if matchTitle(br.body, pattern.Value) {
					score += 0.5
				}
			case "body_key":
				if matchBodyKey(br.body, pattern.Key) {
					score += 0.5
				}
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

// titlePattern extracts the content of the HTML <title> tag.
var titlePattern = regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)

// matchHeader checks if the response headers contain key with a value
// matching the given pattern.
func matchHeader(headers http.Header, key, pattern string) bool {
	value := headers.Get(key)
	if value == "" {
		return false
	}
	if pattern == "" {
		return true
	}
	return strings.Contains(value, pattern)
}

// matchTitle checks if the HTML body contains a <title> tag whose text
// matches the given pattern.
func matchTitle(body []byte, pattern string) bool {
	if pattern == "" {
		return false
	}
	m := titlePattern.FindSubmatch(body)
	if m == nil {
		return false
	}
	return strings.Contains(string(m[1]), pattern)
}

// matchBodyKey checks if the response body contains the given JSON key string.
func matchBodyKey(body []byte, key string) bool {
	if key == "" {
		return false
	}
	return strings.Contains(string(body), `"`+key+`"`)
}
