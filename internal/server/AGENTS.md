# Server Package Guide

HTTP server with Fiber v3, routing, and protocol handlers.

## Structure

```
server/
├── server.go       # Server initialization
├── router.go       # Route definitions
├── event.go        # Event handling
├── http.go         # HTTP helpers
├── func.go         # Server functions
├── mcp.go          # MCP protocol handler
├── chatbot.go      # Chatbot integration
├── modules.go      # Module registration
├── admin.go        # Admin routes
├── database.go     # DB helpers
├── notify.go       # Notification helpers
└── platform.go     # Platform integration
```

## Routes

| Endpoint                 | Handler            |
| ------------------------ | ------------------ |
| `/`                      | Health check       |
| `/livez`                 | Liveness probe     |
| `/readyz`                | Readiness probe    |
| `/startupz`              | Startup probe      |
| `/metrics`               | Prometheus metrics |
| `/oauth/:provider/:flag` | OAuth callback     |
| `/p/:id`                 | Page render        |
| `/form`                  | Form submission    |
| `/page/:id/:flag`        | Page handler       |
| `/agent`                 | Agent data         |
| `/webhook/:flag`         | Webhook handler    |
| `/chatbot/:platform`     | Platform callback  |
| `/mcp/:bot_name`         | MCP protocol       |

## Fiber v3 Patterns

```go
// Middleware
a.Use(middleware...)

// Route groups
g := a.Group("/api")

// Handlers
g.Get("/:id", ctl.handler)
g.Post("/", ctl.create)
```

## MCP Handler

Model Context Protocol at `/mcp/:bot_name`:

- Bearer token auth required
- Routes to bot-specific MCP tools
- Returns protocol Response

## Anti-Patterns

- **Never** block in handlers — use goroutines for long ops
- **Never** use raw `fiber.Ctx` without protocol helpers
- **Always** use protocol error codes for responses
- **Always** validate inputs before processing

## Testing

```bash
go test ./internal/server/...
```
