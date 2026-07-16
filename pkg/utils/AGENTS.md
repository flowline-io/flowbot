# Utilities

General-purpose utility functions with unit tests.

## Structure

```
utils/
├── utils.go, file.go, network.go, slice.go, string.go, ...
├── reexec/   # Self-reexec for upgrades (platform-tagged command_*.go)
├── sets/     # Generic set types (int, string)
└── syncx/    # Generic sync.Map wrapper
```

## Rules

- Prefer a co-located `*_test.go` for each non-trivial `.go` file (platform-tagged `reexec/command_*.go` may share package-level tests).
- Use `utils.CheckSingleton()` for thread-safe single init.
- Use `utils.SignalHandler()` to obtain the signal channel (blocks on SIGTERM/SIGINT; also watches SIGHUP).
- Markdown for browser display: `MarkdownToSafeHTML` (not raw `MarkdownToHTML` + `templ.Raw`).

## Commands

```bash
go test ./pkg/utils/...
```
