# Server Package Guide

HTTP server with Fiber v3, routing, and protocol handlers.

## Anti-Patterns

- **Never** block in handlers — use goroutines for long ops
- **Never** use raw `fiber.Ctx` without protocol helpers
- **Always** use protocol error codes for responses
- **Always** validate inputs before processing

## Testing

```bash
go test ./internal/server/...
```
