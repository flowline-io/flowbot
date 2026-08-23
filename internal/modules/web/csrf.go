package web

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"

	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

const (
	csrfCookieNameLocal = "csrf_"
	csrfCookieNameHost  = "__Host-csrf_"
	csrfHeaderName      = csrf.HeaderName
	csrfFormField       = "csrf_token"
	csrfIdleTimeout     = 30 * time.Minute
	testCSRFToken       = "flowbot-test-csrf-token-32bytes!"
)

var (
	// Fiber CSRF stores this placeholder; validation uses the key.
	csrfDummyValue     = []byte{'+'}
	csrfTokenStore     *csrfMemoryStore
	csrfTokenStoreOnce sync.Once
)

func csrfStore() *csrfMemoryStore {
	csrfTokenStoreOnce.Do(func() {
		csrfTokenStore = newCSRFMemoryStore()
	})
	return csrfTokenStore
}

func csrfCookieName() string {
	return csrfCookieNameFor(authConfig())
}

// HTTPS uses the __Host- prefix so browsers refuse the cookie on plaintext HTTP.
func csrfCookieNameFor(cfg AuthConfig) string {
	if cfg.cookieSecureEnabled() {
		return csrfCookieNameHost
	}
	return csrfCookieNameLocal
}

func newCSRFMiddleware() fiber.Handler {
	secure := authConfig().cookieSecureEnabled()
	handler := csrf.New(csrf.Config{
		CookieName:        csrfCookieNameFor(authConfig()),
		CookiePath:        "/",
		CookieSecure:      secure,
		CookieHTTPOnly:    false,
		CookieSameSite:    "Lax",
		CookieSessionOnly: true,
		IdleTimeout:       csrfIdleTimeout,
		Storage:           csrfStore(),
		TrustedOrigins:    csrfTrustedOrigins(),
		ErrorHandler:      csrfErrorHandler,
		Extractor: extractors.Chain(
			extractors.FromHeader(csrfHeaderName),
			extractors.FromForm(csrfFormField),
		),
		Next: csrfSkip,
	})
	return func(ctx fiber.Ctx) error {
		prepareCSRFOrigin(ctx)
		return handler(ctx)
	}
}

func csrfErrorHandler(c fiber.Ctx, err error) error {
	flog.Warn("web csrf rejected error=%v path=%s method=%s origin=%s host=%s scheme=%s",
		err, c.Path(), c.Method(), c.Get(fiber.HeaderOrigin), c.Host(), c.Scheme())
	return fiber.ErrForbidden
}

func csrfTrustedOrigins() []string {
	raw := strings.TrimSpace(pkgconfig.App.Flowbot.URL)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	return []string{strings.ToLower(u.Scheme) + "://" + csrfHostWithoutDefaultPort(u.Host)}
}

func prepareCSRFOrigin(ctx fiber.Ctx) {
	origin := strings.TrimSpace(ctx.Get(fiber.HeaderOrigin))
	if origin == "" {
		return
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return
	}
	scheme := strings.ToLower(u.Scheme)
	host := csrfHostWithoutDefaultPort(u.Host)
	if scheme == "https" && !strings.EqualFold(ctx.Scheme(), "https") && csrfHostsEqual(host, ctx.Host()) {
		scheme = "http"
		host = ctx.Host()
	}
	normalized := scheme + "://" + host
	if normalized != origin {
		ctx.Request().Header.Set(fiber.HeaderOrigin, normalized)
	}
}

func csrfHostsEqual(a, b string) bool {
	return strings.EqualFold(csrfHostWithoutDefaultPort(a), csrfHostWithoutDefaultPort(b))
}

func csrfHostWithoutDefaultPort(host string) string {
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return host
	}
	if p != "80" && p != "443" {
		return host
	}
	if strings.Contains(h, ":") {
		return "[" + h + "]"
	}
	return h
}

// Login and setup POST always check CSRF. Other unauthenticated mutations skip
// so authenticateWeb can redirect.
func csrfSkip(ctx fiber.Ctx) bool {
	path := string(ctx.Request().URI().Path())
	if !strings.HasPrefix(path, "/service/web") {
		return true
	}
	if csrfExemptMethod(ctx.Method()) {
		return false
	}
	hasSession := ctx.Cookies(webauth.CookieAccessToken) != "" || ctx.Cookies(webauth.CookiePending) != ""
	isAuthPost := ctx.Method() == http.MethodPost && isWebAuthMutationPath(path)
	return !hasSession && !isAuthPost
}

func csrfExemptMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func isWebAuthMutationPath(path string) bool {
	switch {
	case strings.HasSuffix(path, "/login"),
		strings.HasSuffix(path, "/login/2fa"),
		strings.HasSuffix(path, "/setup"),
		strings.HasSuffix(path, "/setup/2fa"),
		strings.HasSuffix(path, "/setup/backup-codes/ack"):
		return true
	default:
		return false
	}
}

// requestIsHTTPS reports whether the request arrived over TLS (or a TLS-terminating proxy).
func requestIsHTTPS(ctx fiber.Ctx) bool {
	if strings.EqualFold(ctx.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return strings.EqualFold(ctx.Protocol(), "https")
}

// requestPublicOrigin returns the absolute site origin for links shown in the UI.
// Prefers config.App.Flowbot.URL; otherwise scheme + host from the request.
func requestPublicOrigin(ctx fiber.Ctx) string {
	if base := strings.TrimRight(strings.TrimSpace(pkgconfig.App.Flowbot.URL), "/"); base != "" {
		return base
	}
	host := strings.TrimSpace(ctx.Host())
	if host == "" {
		return ""
	}
	scheme := "http"
	if requestIsHTTPS(ctx) {
		scheme = "https"
	}
	return scheme + "://" + host
}

func ensureCSRFCookie(ctx fiber.Ctx) (string, error) {
	token := csrf.TokenFromContext(ctx)
	if token == "" {
		return "", fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	return token, nil
}

// Used when document.cookie cannot read a prior Secure cookie on HTTP.
func csrfTokenJSON(ctx fiber.Ctx) error {
	token, err := ensureCSRFCookie(ctx)
	if err != nil {
		return err
	}
	ctx.Set("Cache-Control", "no-store")
	return ctx.JSON(fiber.Map{"token": token})
}

// AttachCSRFForTest sets a fixed CSRF cookie and matching X-Csrf-Token header on req.
// Use for unit and BDD tests that perform state-changing /service/web requests.
// Safe to call after Header.Set("Cookie", ...) — appends the CSRF cookie when missing.
func AttachCSRFForTest(req *http.Request) {
	if req == nil {
		return
	}
	seedCSRFToken(testCSRFToken)
	existing := req.Header.Get("Cookie")
	for _, name := range []string{csrfCookieNameLocal, csrfCookieNameHost} {
		if existing == "" {
			req.AddCookie(&http.Cookie{Name: name, Value: testCSRFToken})
			existing = req.Header.Get("Cookie")
			continue
		}
		if !strings.Contains(existing, name+"=") {
			req.Header.Set("Cookie", existing+"; "+name+"="+testCSRFToken)
			existing = req.Header.Get("Cookie")
		}
	}
	req.Header.Set(csrfHeaderName, testCSRFToken)
}

func seedCSRFToken(token string) {
	csrfStore().put(token, csrfDummyValue, csrfIdleTimeout)
}

// addWebAuth attaches the standard test accessToken cookie and CSRF double-submit pair.
func addWebAuth(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	AttachLocaleForTest(req, i18n.CookieEN)
	AttachCSRFForTest(req)
}

type csrfMemItem struct {
	val []byte
	exp time.Time
}

type csrfMemoryStore struct {
	mu    sync.Mutex
	items map[string]csrfMemItem
}

func newCSRFMemoryStore() *csrfMemoryStore {
	return &csrfMemoryStore{items: make(map[string]csrfMemItem)}
}

func (s *csrfMemoryStore) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *csrfMemoryStore) GetWithContext(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if !item.exp.IsZero() && time.Now().After(item.exp) {
		delete(s.items, key)
		return nil, nil
	}
	return item.val, nil
}

func (s *csrfMemoryStore) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *csrfMemoryStore) SetWithContext(_ context.Context, key string, val []byte, exp time.Duration) error {
	s.put(key, val, exp)
	return nil
}

func (s *csrfMemoryStore) put(key string, val []byte, exp time.Duration) {
	if key == "" || len(val) == 0 {
		return
	}
	item := csrfMemItem{val: append([]byte(nil), val...)}
	if exp > 0 {
		item.exp = time.Now().Add(exp)
	}
	s.mu.Lock()
	s.items[key] = item
	s.mu.Unlock()
}

func (s *csrfMemoryStore) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *csrfMemoryStore) DeleteWithContext(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	return nil
}

func (s *csrfMemoryStore) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (s *csrfMemoryStore) ResetWithContext(_ context.Context) error {
	s.mu.Lock()
	s.items = make(map[string]csrfMemItem)
	s.mu.Unlock()
	return nil
}

func (*csrfMemoryStore) Close() error { return nil }
