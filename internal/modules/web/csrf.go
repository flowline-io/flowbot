package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	csrfCookieName = "csrfToken"
	csrfHeaderName = "X-CSRF-Token"
	csrfFormField  = "csrf_token"
	csrfTokenBytes = 32
	csrfMinLen     = 16
)

// testCSRFToken is a fixed double-submit token used by AttachCSRFForTest.
const testCSRFToken = "flowbot-test-csrf-token-32bytes!"

// generateCSRFToken returns a URL-safe random CSRF token.
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// csrfTokensEqual reports whether a and b are non-empty and equal in constant time.
func csrfTokensEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// requestIsHTTPS reports whether the request arrived over TLS (or a TLS-terminating proxy).
func requestIsHTTPS(ctx fiber.Ctx) bool {
	if strings.EqualFold(ctx.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return strings.EqualFold(ctx.Protocol(), "https")
}

// csrfCookieSecure decides the Secure flag for the CSRF cookie.
// CSRF must be readable by JS on local HTTP; only set Secure when the request is
// actually HTTPS (or forwarded as such) and cookie_secure is enabled.
func csrfCookieSecure(ctx fiber.Ctx) bool {
	return authConfig().cookieSecureEnabled() && requestIsHTTPS(ctx)
}

// setCSRFCookie writes the non-HttpOnly csrfToken cookie (readable by JS for double-submit).
func setCSRFCookie(ctx fiber.Ctx, token string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		HTTPOnly: false,
		SameSite: "Lax",
		Secure:   csrfCookieSecure(ctx),
		Path:     "/",
		MaxAge:   86400,
	})
}

// ensureCSRFCookie returns the CSRF token for this request, issuing a new cookie when needed.
func ensureCSRFCookie(ctx fiber.Ctx) (string, error) {
	existing := ctx.Cookies(csrfCookieName)
	if len(existing) >= csrfMinLen {
		return existing, nil
	}
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}
	setCSRFCookie(ctx, token)
	return token, nil
}

// readCSRFSubmission returns the CSRF token from header or form field.
func readCSRFSubmission(ctx fiber.Ctx) string {
	if h := ctx.Get(csrfHeaderName); h != "" {
		return h
	}
	return ctx.FormValue(csrfFormField)
}

// csrfExemptMethod reports whether the HTTP method does not require CSRF validation.
func csrfExemptMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// csrfMiddleware issues CSRF cookies on safe requests and validates double-submit on mutations
// under /service/web. Static assets are not covered by this mount path.
// Unauthenticated non-login mutations skip CSRF so authenticateWeb can redirect to login;
// cookie sessions and login POST always require a matching token.
func csrfMiddleware(ctx fiber.Ctx) error {
	path := string(ctx.Request().URI().Path())
	if !strings.HasPrefix(path, "/service/web") {
		return ctx.Next()
	}
	if csrfExemptMethod(ctx.Method()) {
		if _, err := ensureCSRFCookie(ctx); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
		}
		return ctx.Next()
	}
	hasSession := ctx.Cookies("accessToken") != ""
	isLoginPost := ctx.Method() == http.MethodPost && strings.HasSuffix(path, "/login")
	if !hasSession && !isLoginPost {
		return ctx.Next()
	}
	cookieTok := ctx.Cookies(csrfCookieName)
	submitTok := readCSRFSubmission(ctx)
	if !csrfTokensEqual(cookieTok, submitTok) {
		return fiber.NewError(fiber.StatusForbidden, "invalid CSRF token")
	}
	return ctx.Next()
}

// csrfTokenJSON returns the current CSRF token as JSON and ensures the cookie is set.
// Used by the browser when document.cookie cannot read a prior Secure cookie on HTTP.
func csrfTokenJSON(ctx fiber.Ctx) error {
	token, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	setCSRFCookie(ctx, token)
	return ctx.JSON(fiber.Map{"token": token})
}

// AttachCSRFForTest sets a fixed CSRF cookie and matching X-CSRF-Token header on req.
// Use for unit and BDD tests that perform state-changing /service/web requests.
// Safe to call after Header.Set("Cookie", ...) — appends csrfToken to the Cookie header.
func AttachCSRFForTest(req *http.Request) {
	if req == nil {
		return
	}
	existing := req.Header.Get("Cookie")
	if existing == "" {
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: testCSRFToken})
	} else if !strings.Contains(existing, csrfCookieName+"=") {
		req.Header.Set("Cookie", existing+"; "+csrfCookieName+"="+testCSRFToken)
	}
	req.Header.Set(csrfHeaderName, testCSRFToken)
}

// addWebAuth attaches the standard test accessToken cookie and CSRF double-submit pair.
func addWebAuth(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	AttachCSRFForTest(req)
}
