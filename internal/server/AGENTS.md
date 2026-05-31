# Server Package

HTTP server with Fiber v3, routing, and protocol handlers.

## Rules

- Never block in handlers — use goroutines for long ops
- Use protocol helpers (`protocol.NewFailedResponse`, `protocol.NewSuccessResponse`) for structured responses; map `types.Err*` sentinels in `error.go` to HTTP status codes
- Always validate inputs before processing

## Testing

```bash
go test ./internal/server/...
```
