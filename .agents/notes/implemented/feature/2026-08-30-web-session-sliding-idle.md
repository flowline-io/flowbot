# Agent Note: Web UI sliding 24h idle session

Status: implemented

## Problem

Web UI `accessToken` cookies used a fixed 24h lifetime from login. An operator who kept using the console was logged out at that wall-clock time. Navbar copy showed a remaining-time countdown that froze at page load and would disagree with a sliding policy.

## Decision

Full browser web sessions (`kind=full`, `topic=web`, cookie `accessToken`) use a sliding idle TTL of `webauth.FullSessionTTL` (24h). `slideFullSessionMiddleware` on `/service/web` (after locale and CSRF) pushes the same token's `expired_at` and cookie `Max-Age` to now + 24h when remaining time is at most `FullSessionTTL - SessionSlideThrottle` (5 minutes). Persist failures are logged and the request still succeeds. API tokens, pending 2FA/enroll cookies, and header-only credentials are not slid. Navbar `session-badge` shows the i18n label `session.ttl_label` (`24h session` / `24 小时会话`), not a countdown. `route.throttledUpdateLastUsed` writes params via `SetParams` so last-used updates cannot shorten expiry.

## Alternatives considered

- **Hard 24h cap plus a shorter idle window.** Rejected: homelab ops leave a tab open; the requested model is idle 24h from last authenticated request, including badge polls.
- **Rotate the token on each slide.** Rejected: concurrent 20s badge polls would race and 401.
- **Slide inside `route.Authorize` for every access token.** Rejected: would rewrite composer/API token TTL and pending sessions.
- **Slide only in `authenticateWeb`.** Rejected: pipeline and function handlers never call it, so those `/service/web` requests would not extend the session.
- **Poll `session-badge` to keep the countdown live.** Rejected: a third navbar poller competes for HTTP/1.1 connection slots; sliding makes a ticking remainder misleading.
- **YAML for TTL and throttle.** Rejected: login lifetime is a security constant, not an environment knob.

## Consequences

- Any `/service/web` request that still carries a slidable cookie extends the idle clock, including pipeline/function routes and 20s inbox/approval badge polls. Closing the tab starts the 24h idle clock from the last persist.
- Badge polls do not add a `session-badge` poller.
- Concurrent `last_used_at` writes no longer clobber a just-slid `expired_at`.

## Verification

- `go test ./pkg/webauth/ -run 'TestIsSlidableFullSession|TestShouldSlideFullSession'` — full+web is slidable; pending/other topic is not; remaining 24h skips, remaining ≤ 23h55m slides, expired does not resurrect.
- `go test ./pkg/route/ -run TestThrottledUpdateLastUsed_PreservesExpiry` — missing, stale, and fresh `last_used_at` leave `expired_at` unchanged.
- `go test ./internal/store/ -run 'TestOAuthFormAndParameter/parameter update params keeps expired_at'` — `ParameterUpdateParams` changes params and keeps expiry; missing flag is not found.
- `go test ./internal/modules/web/ -run TestSlideFullSession` — 1h remaining cookie slides to ~24h and `max-age=86400`; fresh 24h remaining skips; header-only and non-web topic skip; `GET /pipelines` (no `authenticateWeb`) still slides.
- `go test ./pkg/views/partials/ -run TestSessionBadge` — SVG size, username, and `24h session` label render.
