# Utilities

General-purpose utility functions with unit tests.

## Structure

```
utils/
├── utils.go, file.go, json.go, network.go, slice.go, string.go, ...
├── reexec/   # Self-reexec for upgrades
├── sets/     # Generic set types (int, string)
└── syncx/    # Generic sync.Map wrapper
```

## Rules

- Every `.go` file must have a corresponding `*_test.go`.
- Use `utils.NewSingleton()` for thread-safe single init.
- Use `utils.WaitSignal()` to block on SIGTERM/SIGINT.

## Commands

```bash
go test ./pkg/utils/...
```
