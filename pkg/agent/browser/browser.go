// Package browser provides a CDP-backed browser session for agent tools.
package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/utils"
)

// Driver selects CDP backend quirks.
type Driver string

const (
	// DriverLightpanda is the Lightpanda headless browser CDP server.
	DriverLightpanda Driver = "lightpanda"
	// DriverPlaywright is Chromium exposed via remote-debugging CDP
	// (compose profile playwright-cdp), not the Playwright Test runner.
	DriverPlaywright Driver = "playwright"

	defaultTimeout  = 30 * time.Second
	defaultEndpoint = "ws://127.0.0.1:9222"
	// MaxScreenshotBytes caps PNG bytes returned by Screenshot.
	MaxScreenshotBytes = 2 << 20
	maxSnapshotChars   = 100_000
	refAttr            = "data-flowbot-ref"
	defaultWaitMs      = 500
	safeBlankURL       = "about:blank"
)

// Config configures a browser session.
type Config struct {
	Driver       Driver
	Endpoint     string
	AllowPrivate bool
	AllowHosts   []string
	Timeout      time.Duration
}

// Normalize fills defaults and validates driver/endpoint.
func (c Config) Normalize() (Config, error) {
	out := c
	if out.Driver == "" {
		out.Driver = DriverLightpanda
	}
	switch out.Driver {
	case DriverLightpanda, DriverPlaywright:
	default:
		return Config{}, fmt.Errorf("browser: unknown driver %q", out.Driver)
	}
	out.Endpoint = strings.TrimSpace(out.Endpoint)
	if out.Endpoint == "" {
		out.Endpoint = defaultEndpoint
	}
	if out.Timeout <= 0 {
		out.Timeout = defaultTimeout
	}
	return out, nil
}

// Quirks returns driver-specific behavior flags.
func (c Config) Quirks() Quirks {
	switch c.Driver {
	case DriverPlaywright:
		return Quirks{PreferNetworkIdle: true}
	default:
		return Quirks{}
	}
}

// Quirks holds per-driver capability differences.
type Quirks struct {
	PreferNetworkIdle bool
}

type sessionCtxKey struct{}

// WithSession attaches a Session to ctx for browser tools.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, s)
}

// FromContext returns the Session attached to ctx, if any.
func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionCtxKey{}).(*Session)
	return s, ok && s != nil
}

// Session is a per-run isolated browser context with serialized operations.
type Session struct {
	cfg Config

	mu     sync.Mutex
	closed bool
	page   cdpPage
	refs   map[string]struct{}
}

// NewSession creates a lazy session. CDP connects on first operation.
func NewSession(cfg Config) (*Session, error) {
	normalized, err := cfg.Normalize()
	if err != nil {
		return nil, err
	}
	return &Session{
		cfg:  normalized,
		refs: make(map[string]struct{}),
	}, nil
}

// Config returns the session configuration.
func (s *Session) Config() Config { return s.cfg }

// Close tears down the CDP browser context.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.refs = make(map[string]struct{})
	if s.page == nil {
		return nil
	}
	err := s.page.Close()
	s.page = nil
	return err
}

// Navigate loads url after URL-gate checks and clears refs.
func (s *Session) Navigate(ctx context.Context, rawURL string) (string, error) {
	if err := AssertNavigateURL(rawURL, s.cfg.AllowPrivate, s.cfg.AllowHosts); err != nil {
		return "", err
	}
	return withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		finalURL, err := page.Navigate(ctx, rawURL)
		if err != nil {
			return "", err
		}
		s.clearRefsLocked()
		if finalURL != "" {
			if err := AssertNavigateURL(finalURL, s.cfg.AllowPrivate, s.cfg.AllowHosts); err != nil {
				_, _ = page.Navigate(ctx, safeBlankURL)
				s.clearRefsLocked()
				return "", fmt.Errorf("blocked final url: %w", err)
			}
		}
		return finalURL, nil
	})
}

// Snapshot returns a compact interactive tree with opaque refs.
func (s *Session) Snapshot(ctx context.Context) (string, error) {
	return withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		tree, refs, err := page.Snapshot(ctx)
		if err != nil {
			return "", err
		}
		s.refs = refs
		if len(tree) > maxSnapshotChars {
			tree = tree[:maxSnapshotChars] + "\n...(snapshot truncated)"
		}
		return tree, nil
	})
}

// Click activates the element identified by ref from the last Snapshot.
func (s *Session) Click(ctx context.Context, ref string) error {
	_, err := withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		if err := s.requireRefLocked(ref); err != nil {
			return "", err
		}
		return "", page.Click(ctx, ref)
	})
	return err
}

// Type focuses ref and types text.
func (s *Session) Type(ctx context.Context, ref, text string) error {
	_, err := withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		if err := s.requireRefLocked(ref); err != nil {
			return "", err
		}
		return "", page.Type(ctx, ref, text)
	})
	return err
}

// Scroll scrolls the page by dx, dy pixels.
func (s *Session) Scroll(ctx context.Context, dx, dy int) error {
	_, err := withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		return "", page.Scroll(ctx, dx, dy)
	})
	return err
}

// Wait pauses or waits for network idle depending on driver quirks.
func (s *Session) Wait(ctx context.Context, ms int) error {
	if ms <= 0 {
		ms = defaultWaitMs
	}
	_, err := withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		return "", page.Wait(ctx, ms, s.cfg.Quirks())
	})
	return err
}

// Screenshot captures a PNG of the current page.
func (s *Session) Screenshot(ctx context.Context) ([]byte, error) {
	var png []byte
	_, err := withLock(ctx, s, func(ctx context.Context, page cdpPage) (string, error) {
		data, err := page.Screenshot(ctx)
		if err != nil {
			return "", err
		}
		if len(data) > MaxScreenshotBytes {
			return "", fmt.Errorf("screenshot exceeds %d bytes", MaxScreenshotBytes)
		}
		png = data
		return "", nil
	})
	return png, err
}

func (s *Session) requireRefLocked(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("ref is required")
	}
	if _, ok := s.refs[ref]; !ok {
		return fmt.Errorf("unknown or stale ref %q; call browser_snapshot again", ref)
	}
	return nil
}

func (s *Session) clearRefsLocked() {
	s.refs = make(map[string]struct{})
}

func withLock(ctx context.Context, s *Session, fn func(context.Context, cdpPage) (string, error)) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", errors.New("browser session closed")
	}
	if err := s.ensurePageLocked(); err != nil {
		return "", err
	}
	opCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	return fn(opCtx, s.page)
}

func (s *Session) ensurePageLocked() error {
	if s.page != nil {
		return nil
	}
	// Connection outlives per-op timeouts; Close tears it down.
	page, err := openCDPPage(context.Background(), s.cfg)
	if err != nil {
		return err
	}
	s.page = page
	return nil
}

// AssertNavigateURL validates a navigation target against SSRF policy.
// allow_hosts is an explicit exception list. When allowPrivate is true, RFC1918
// private addresses are permitted, but loopback, link-local, and cloud-metadata
// hosts remain blocked unless listed in allow_hosts.
func AssertNavigateURL(raw string, allowPrivate bool, allowHosts []string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("only http and https URLs are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errors.New("url host is required")
	}
	if hostAllowed(host, allowHosts) {
		return nil
	}
	if allowPrivate {
		return assertPrivateAllowedStillSafe(u)
	}
	return utils.AssertPublicHTTPURL(u, false)
}

func assertPrivateAllowedStillSafe(u *url.URL) error {
	host := strings.ToLower(u.Hostname())
	if isSensitiveHostname(host) {
		return fmt.Errorf("host %q is not allowed", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isSensitiveIP(ip) {
			return fmt.Errorf("address %s is not allowed", ip)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	for _, ip := range ips {
		if isSensitiveIP(ip) {
			return fmt.Errorf("host %q resolves to blocked address %s", host, ip)
		}
	}
	return nil
}

func isSensitiveHostname(host string) bool {
	switch host {
	case "localhost", "metadata.google.internal":
		return true
	}
	return strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local")
}

func isSensitiveIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
		return true
	}
	return false
}

func hostAllowed(host string, allow []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, a := range allow {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if _, cidr, err := net.ParseCIDR(a); err == nil {
			if ip := net.ParseIP(host); ip != nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// cdpPage is the CDP-backed page operations used by Session.
type cdpPage interface {
	Navigate(ctx context.Context, rawURL string) (finalURL string, err error)
	Snapshot(ctx context.Context) (tree string, refs map[string]struct{}, err error)
	Click(ctx context.Context, ref string) error
	Type(ctx context.Context, ref, text string) error
	Scroll(ctx context.Context, dx, dy int) error
	Wait(ctx context.Context, ms int, quirks Quirks) error
	Screenshot(ctx context.Context) ([]byte, error)
	Close() error
}
