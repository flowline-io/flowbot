# Providers Guide

Third-party service integrations with standardized OAuth and API patterns.

## Structure

```
providers/
├── providers.go      # Base interfaces and utilities
├── <service>/
│   ├── provider.go   # Provider implementation
│   ├── types.go      # Service-specific types
│   └── client.go     # API client (optional)
```

## Provider Types

| Category       | Providers                                    |
| -------------- | -------------------------------------------- |
| Communication  | slack, email                                 |
| Development    | github, gitea, drone                         |
| Productivity   | kanboard, n8n                                |
| Finance        | fireflyiii                                   |
| Infrastructure | cloudflare, adguard, uptimekuma              |
| Media          | transmission, miniflux, archivebox, karakeep |
| Storage        | dropbox                                      |
| Other          | slash                                        |

## OAuth Pattern

Providers implementing OAuth:

```go
type OAuthProvider interface {
    GetAuthorizeURL() string
    GetAccessToken(ctx fiber.Ctx) (types.KV, error)
}
```

## Configuration

Access via `flowbot.yaml`:

```yaml
providers:
  github:
    client_id: "xxx"
    client_secret: "xxx"
  slack:
    bot_token: "xoxb-..."
```

## Implementation Pattern

1. Create `pkg/providers/<name>/provider.go`
2. Implement service interface
3. Register in provider list
4. Add config struct

## Anti-Patterns

- **Never** hardcode credentials
- **Never** ignore rate limits
- **Always** handle API errors gracefully
- **Always** use context for timeouts

## Testing

- Use `pkg/providers/<name>/` package for tests
- Mock HTTP clients with `httptest`
- Test auth flows separately from API calls
