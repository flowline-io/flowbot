# Utilities

General-purpose helpers with co-located tests.

## Entry points

Package root (`utils.go`, `file.go`, …) plus `reexec/`, `sets/`, `syncx/`.

## Non-obvious rules

- Prefer `*_test.go` per non-trivial `.go` file (`reexec/command_*.go` may share package tests)
- `CheckSingleton()` for thread-safe single init; `SignalHandler()` for SIGTERM/SIGINT (+ SIGHUP)
- Browser markdown: `MarkdownToSafeHTML` (not raw `MarkdownToHTML` + `templ.Raw`)

## Testing

```bash
go test ./pkg/utils/...
```
