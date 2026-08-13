# Providers Guide

Third-party API/OAuth clients. Configure under `flowbot.yaml` → `providers.<name>`. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md).

## Entry points

- Shared: `providers.go` (`GetConfig`, `RegisterOAuthProvider`, …)
- Reference: `example/` (`GetClient` / `NewXxx`, OAuth methods, httptest tests)
- Per service: `<service>/` — preferred `<service>.go` + optional `types.go`

## Boundaries

- OAuth production providers export `Register()` and wire via `fx.Invoke` in `internal/server/providers.go` (github, slack, dropbox). `example` shows OAuth methods but does **not** export `Register()`
- Token/API-key providers skip OAuth `Register` / fx wiring
- Never hardcode credentials; respect rate limits; use context timeouts
- Persist OAuth tokens via injected `OAuthTokenStore` (`SetOAuthTokenStore` from `internal/server`) ([pkg-boundaries.md](../../docs/architecture/pkg-boundaries.md))
- `GetOrRefreshToken` must not touch `gen.OAuth`; mapping belongs in the server adapter

## Testing

Package tests mock with `httptest`; separate auth flows from API calls. Layers: [docs/testing/README.md](../../docs/testing/README.md).
