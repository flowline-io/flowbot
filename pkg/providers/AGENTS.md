# Providers Guide

Third-party API/OAuth clients. Configure under `flowbot.yaml` → `providers.<name>`.

## Entry points

- Shared: `providers.go` (`GetConfig`, `RegisterOAuthProvider`, …)
- Reference: `example/` (`GetClient` / `NewXxx`, OAuth methods, httptest tests)
- Per service: `<service>/` — preferred `<service>.go` + optional `types.go`

## Boundaries

- OAuth production providers export `Register()` and wire via `fx.Invoke` in `internal/server/providers.go` (github, slack, dropbox). `example` shows OAuth methods but does **not** export `Register()`
- Token/API-key providers skip OAuth `Register` / fx wiring
- Never hardcode credentials; respect rate limits; use context timeouts

## Testing

Mock with `httptest`; separate auth flows from API calls.
