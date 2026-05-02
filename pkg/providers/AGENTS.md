# Providers Guide

Third-party service integrations with standardized OAuth and API patterns.

## Structure

```
providers/
├── providers.go   # Base interfaces
└── <service>/
    ├── provider.go # Provider implementation
    └── types.go    # Service-specific types
```

## Patterns

- Configure via `flowbot.yaml` under `providers.<name>`.
- OAuth providers implement `GetAuthorizeURL` / `GetAccessToken`.
- Register new providers in the provider list.

## Rules

- Never hardcode credentials
- Never ignore rate limits
- Always handle API errors gracefully
- Always use context for timeouts

## Testing

- Mock HTTP clients with `httptest`
- Test auth flows separately from API calls
