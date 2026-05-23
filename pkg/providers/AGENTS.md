# Providers Guide

Third-party service integrations with standardized OAuth and API patterns.

## Structure

```
providers/
├── providers.go        # Base interfaces (OAuthProvider, GetConfig, RedirectURI)
├── example/            # Reference implementation — follow this for new providers
│   ├── example.go      # Provider struct, GetClient(), NewXxx(), CRUD, OAuth
│   ├── types.go        # Request/response/webhook-payload types
│   └── example_test.go # TDD unit tests (table-driven, httptest mock)
└── <service>/
    ├── <service>.go     # Provider implementation (file name = package name)
    ├── types.go         # Service-specific types
    └── <service>_test.go
```

## Patterns

- **Reference implementation**: When creating or modifying a provider, reference `pkg/providers/example/` for file structure, naming, and code style.
- Configure via `flowbot.yaml` under `providers.<name>`.
- OAuth providers implement `GetAuthorizeURL` / `GetAccessToken`. See `example/example.go` for the complete OAuth reference.
- Register new providers in the provider list.
- Constructor pattern: `GetClient()` reads config via `providers.GetConfig()` then calls `NewXxx()`. See `example/example.go` for the full pattern.

## Rules

- Never hardcode credentials
- Never ignore rate limits
- Always handle API errors gracefully
- Always use context for timeouts

## Testing

- Mock HTTP clients with `httptest`
- Test auth flows separately from API calls
