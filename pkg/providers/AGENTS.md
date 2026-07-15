# Providers Guide

Third-party service integrations with standardized OAuth and API patterns.

## Structure

```
providers/
├── providers.go        # OAuthProvider, GetConfig, RedirectURI, RegisterOAuthProvider, …
├── example/            # Reference implementation — follow this for new providers
│   ├── example.go      # Provider struct, GetClient(), NewXxx(), CRUD, OAuth methods
│   ├── types.go        # Request/response/webhook-payload types
│   └── example_test.go # TDD unit tests (table-driven, httptest mock)
└── <service>/
    ├── <service>.go     # Provider implementation (preferred; name may vary, e.g. adguard_home.go)
    ├── types.go         # Service-specific types (optional for tiny clients)
    └── <service>_test.go
```

## Patterns

- **Reference implementation**: When creating or modifying a provider, reference `pkg/providers/example/` for file structure, naming, and code style.
- Configure via `flowbot.yaml` under `providers.<name>`.
- OAuth providers implement `GetAuthorizeURL` / `GetAccessToken`. See `example/example.go` for the OAuth method reference.
- Production OAuth providers also export `Register()` → `providers.RegisterOAuthProvider(ID, factory)` and wire via `fx.Invoke` in `internal/server/providers.go` (currently: github, slack, dropbox). The `example` package demonstrates OAuth methods but does **not** export `Register()` / fx wiring.
- Constructor pattern: `GetClient()` reads config via `providers.GetConfig()` then calls `NewXxx()`. See `example/example.go` for the full pattern.
  - `GetClient()` may return `(*T, error)` when config validation is required; the example returns `*T` directly for simplicity.
- Not all providers implement OAuth — token/API-key providers skip `Register()` and `fx.Invoke` wiring.

## Rules

- Never hardcode credentials
- Never ignore rate limits
- Always handle API errors gracefully
- Always use context for timeouts

## Testing

- Mock HTTP clients with `httptest`
- Test auth flows separately from API calls
