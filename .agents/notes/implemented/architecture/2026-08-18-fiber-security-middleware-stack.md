# Agent Note: Fiber official HTTP security stack

Status: implemented

## Problem

Fiber v3 documents a security middleware order and production defaults (Helmet → CORS → CSRF, HSTS, CSRF `__Host-` cookies, CSRF header extractors). Flowbot already ran Fiber v3.5.0 but used a hand-rolled header middleware and double-submit CSRF cookie named `csrfToken`, so those defaults were not the live contract.

## Decision

The HTTP app mounts Fiber Helmet before CORS. CSRF is Fiber `csrf` middleware, path-mounted on `/service/web` after CORS.

Helmet sets `X-Frame-Options: DENY` (stricter than Helmet’s `SAMEORIGIN`), the existing CSP (no `'unsafe-eval'`), `Referrer-Policy: no-referrer`, a camera/microphone/geolocation `Permissions-Policy`, and HSTS `max-age=63072000; includeSubDomains`. HSTS preload is off (homelab hostnames must not be submitted to the preload list). `Cross-Origin-Embedder-Policy` is `unsafe-none` so blob/data UI assets keep working. `Cross-Origin-Resource-Policy` is `cross-origin` so chat-agent signed media can be fetched by LLM providers.

Helmet only emits HSTS when `c.Secure()` is true. `hstsOverlayMiddleware` still sets the same HSTS value when `config.App.ShouldSendHSTS()` is true (TLS-terminating reverse proxy / `cookie_secure`) even if Fiber saw plaintext HTTP.

CSP is stripped on `/swagger/` only.

CORS `AllowHeaders` includes `X-Csrf-Token`. Empty `allow_origins` still does not reflect Origin.

Fiber CSRF uses the official production cookie flags (`CookieSecure`, `SameSite=Lax`, `CookieHTTPOnly: false`, `CookieSessionOnly`, `CookiePath=/`). Cookie name is `__Host-csrf_` when `modules.web.auth.cookie_secure` is enabled (the default) and `csrf_` when it is explicitly false so local HTTP can still set the cookie. Tokens are taken from `X-Csrf-Token` then the `csrf_token` form field (login HTML without JS). Unauthenticated non-login mutations still skip CSRF so `authenticateWeb` can redirect. `EnableIPValidation` is always on in `fiber.Config`. TLS-terminating proxy Origin: [csrf-login-403-tls-proxy](../bug-fix/2026-08-23-csrf-login-403-tls-proxy.md).

## Alternatives considered

- **Drop-in `helmet.New()` / `csrf.New()` defaults.** Helmet’s default `X-Frame-Options` is `SAMEORIGIN`, HSTS is off, and `Cross-Origin-Embedder-Policy: require-corp` plus `Cross-Origin-Resource-Policy: same-origin` would break signed media and parts of the Web UI. CSRF defaults use an insecure cookie name and `CookieSecure: false`.
- **Keep hand-rolled CSRF and only add Helmet.** Official alignment is the three-middleware stack; Fiber CSRF also stores tokens and checks Origin / `Sec-Fetch-Site`.
- **HSTS preload.** Fiber’s blog enables preload. Preload is a public-suffix commitment that does not fit self-hosted / changing homelab names.

## Consequences

Browsers receive a new CSRF cookie name; a GET of any `/service/web` page issues it. Local HTTP must keep `cookie_secure: false`. Tests seed Fiber CSRF storage via `AttachCSRFForTest`.

## Verification

`TestSecurityHeadersMiddleware_HSTS` asserts Helmet `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, the camera/microphone/geolocation Permissions-Policy, `Cross-Origin-Embedder-Policy: unsafe-none`, `Cross-Origin-Resource-Policy: cross-origin`, CSP without `'unsafe-eval'`, overlay HSTS `max-age=63072000; includeSubDomains`, and swagger CSP stripping. `TestHelmetEmitsHSTSWhenSecure` asserts Helmet emits that same HSTS value (no preload) when `c.Secure()` is true via a trusted `X-Forwarded-Proto`. `TestHTTPServerSecurityConfig` asserts `EnableIPValidation`. `TestCORSAllowsCSRFHeader` and `TestCORSAllowOriginsWhitelist` cover CORS. `TestCSRFMiddleware` covers login CSRF and unauthenticated account mutations skipping CSRF. `TestCSRFCookieFlags` asserts production cookie flags. `go tool task lint` is the style gate.
