# Agent Note: Security audit hardening (auth, SSRF, templates, webhooks)

Status: implemented

## Problem

A full-codebase security review found privilege escalation from `pipeline:*` API tokens into the Web UI (including minting `admin:*`), open redirects via `safeNext`, incomplete `web_fetch` SSRF checks, notification templates that could read process env via Sprig `env`, webhook/function query tokens leaking into logs, insecure reference defaults, login rate-limit fail-open on Redis errors, `/agent` accepting any valid token without scope, Web toast leakage of `err.Error()`, and OAuth tokens stored plaintext at rest.

## Decision

Ship a coordinated hardening pass:

1. `/service/web` requires `admin:*`; `authenticateWeb` accepts only `kind=full` sessions; token minting cannot elevate beyond caller scopes (`auth.CanGrantScopes`).
2. Login and inbox redirects share `safeServiceWebRedirectURL` (parse URL; reject `\`, host, userinfo, opaque, non-`/service/web` paths).
3. Shared `utils.AssertPublicHTTPURL` (DNS + private/metadata blocks) used by `web_fetch` and `core.http_request`.
4. Notify templates use `sprig.HermeticTxtFuncMap()` (no `env` / `expandenv` / `getHostByName`).
5. Pipeline/function/workflow webhook auth accepts header or HMAC only; UI URLs omit `?token=`; curl examples use `X-Webhook-Token`.
6. Reference config defaults to loopback listen, platforms disabled, no shipped plaintext web password; Slack gains `required_if=Enabled`.
7. Login rate limiter fails closed on Redis errors; `POST /agent` requires `admin:*`.
8. User-facing Web toast/form errors for channel load, send, and channel test use `clientSafeErrorMessage`; audit/DB records may still store `err.Error()`.
9. OAuth `token` / `refresh_token` sealed with the web auth AES-GCM key (`fbenc1.` prefix); legacy plaintext rows still open until rewritten; sealed values fail closed when the encryptor is nil. Provider YAML secrets remain file/env-based (homelab config model).

## Alternatives considered

- Introduce a dedicated `web:ui` scope instead of `admin:*` for Web routes — deferred; full sessions already carry `admin:*`, and a new scope would require token/UI migration without reducing blast radius for this release.
- Keep query tokens with a deprecation window — rejected under pre-1.0 foundation-over-shims; header/HMAC is the supported contract.
- Encrypt provider `vendors.*` YAML at rest — out of scope; recommend `${ENV}` expansion and file permissions instead of inventing a second secret store.

## Consequences

- Existing integrations that called pipeline/function webhooks with `?token=` must switch to `X-Webhook-Token` or HMAC. Migration UX (UI labels, curl copy, audit warnings): [webhook-query-token-migration-ux](../feature/2026-09-11-webhook-query-token-migration-ux.md).
- API tokens with only `pipeline:*` can no longer drive `/service/web` or mint broader scopes.
- Docker/self-host configs must set `listen: ":6060"` (or `0.0.0.0:6060`) when publishing a container port; loopback is the secure local default.
- OAuth rows written after web Init are ciphertext; backups still need key material (`encryption_key` / key file).

## Verification

- Package tests cover auth scopes, redirects, urlguard, hermetic templates, webhook auth, OAuth seal/open, and login rate-limit fail-closed.
- BDD web page fixtures seed `admin:*` + `kind=full` for authenticated `/service/web` requests; `bddWebScopesUser` (`pipeline:run`) is only for Authorize denial cases (e.g. events page non-admin).
- `go tool task lint` passes on the hardening change set.
