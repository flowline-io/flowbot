# Media Package

Media upload/download handlers, handler registry, signed URLs, and file metadata access via injected store.

## Entry points

- Core: `media.go` (`Handler`), `registry.go` (`RegisterHandler`, `UseHandler`, `FileMetaStore`), `accessor.go`, `sign.go`
- Handlers: `fs/`, `minio/` — register via exported `Register()` wired from `internal/server/media.go`

## Boundaries

- Never import `internal/store` — use injected `FileMetaStore`; register handlers with `media.RegisterHandler` / `UseHandler`
- File metadata APIs take `*types.FileDef` only; no `*gen.*`
- Wire `SetFileMetaStore` from `internal/server` at process start

## Testing

```bash
go test ./pkg/media/...
```
