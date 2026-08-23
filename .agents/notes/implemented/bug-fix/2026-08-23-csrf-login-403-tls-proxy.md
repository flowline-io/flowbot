# Agent Note: Login POST 403 behind TLS-terminating proxy

Status: implemented

## Problem

A browser POST to `/service/web/login` on a public HTTPS origin (for example `https://idev.ink`) returns 403 Forbidden in ~1ms. Fiber CSRF compares `Origin` to `c.Scheme()+Host`. `TrustProxy` is off unless `http.trusted_proxies` is set, so the process scheme stays `http` while the browser sends `Origin: https://<host>`. Unit tests omitted `Origin`, so HTTP Fiber skipped that check and login looked fine.

## Decision

CSRF middleware strips default Origin ports (`:443` / `:80`), rewrites a same-host `https` Origin to the plaintext equivalent before Fiber CSRF runs, and sets `TrustedOrigins` from `flowbot.url` when Host is the internal listener. IPv6 hosts keep brackets after port split. Rejections log the underlying Fiber CSRF error at warn; the client still sees 403. Cross-site Origin remains forbidden.

## Alternatives considered

- **Enable `TrustProxy` for all private networks when `tls_behind_proxy` is set.** Would also fix client IP, but lets any Docker-network peer spoof `X-Forwarded-For` unless `trusted_proxies` is explicit.
- **Skip Fiber Origin checks.** Drops a CSRF layer Fiber documents as required for unsafe methods.
- **TrustedOrigins only.** Fixes Host mismatch when `flowbot.url` is set; leaves same-host HTTPS Origin vs plaintext Fiber broken when the URL is empty.

## Consequences

Login works behind a TLS terminator without `trusted_proxies`. `flowbot.url` is still required when the proxy does not forward the public `Host`. Warn logs include `origin` / `host` / `scheme` so the next 403 is diagnosable.

## Verification

`TestCSRFLoginPOSTHTTPSOriginBehindProxy` asserts same-host HTTPS Origin, `:443` Origin (same Host and vs internal Host via `flowbot.url`), IPv6 `[::1]:443`, and cross-site 403. `TestCSRFTrustedOrigins` parses `flowbot.url` including default-port strip. `TestCSRFHostWithoutDefaultPort` covers IPv6 brackets. `TestCSRFMiddleware` still covers token match / skip. `go test ./internal/modules/web/ -run TestCSRF`.
