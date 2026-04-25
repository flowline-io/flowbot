# Utilities Guide

General-purpose utility functions with required unit test coverage.

**Rule**: Every `.go` file in this directory (and subdirectories) MUST have a corresponding `_test.go` file.

## Structure

```
utils/
├── utils.go              # General helpers (int, map, slice utilities)
├── file.go               # File operations
├── host.go               # Host/metadata detection
├── json.go               # JSON parsing helpers
├── network.go            # Network utilities
├── reflect.go            # Reflection helpers
├── resty.go              # HTTP client wrapper (go-resty)
├── signal.go             # OS signal handling
├── singleton.go          # Singleton pattern
├── slice.go              # Slice manipulation
├── string.go             # String utilities
├── unsafe.go             # Unsafe operations
├── reexec/               # Self-reexec for upgrades
│   ├── rexec.go          # Core logic
│   ├── command_linux.go  # Linux reexec
│   ├── command_unix.go   # Unix reexec
│   └── command_unsupported.go  # No-op fallback
├── sets/                 # Generic set types
│   ├── int.go            # Int set
│   └── string.go         # String set
└── syncx/                # Concurrency primitives
    └── map.go            # Generic sync.Map wrapper
```

## Testing Rule

**Mandatory**: Every `.go` file must have a corresponding `_test.go` file with `require`/`assert` from testify.

```bash
go test ./pkg/utils/...     # All utils tests (must pass)
go test -run ^TestFoo$ ./pkg/utils  # Specific test
```

## Key Patterns

**Singleton**: `utils.NewSingleton(func() (any, error) { ... })` — thread-safe single init.

**Reexec**: `reexec.Init()` returns true if daemon should exit (self-upgrade). Platform-specific implementations in `command_linux.go`, `command_unix.go`.

**Signal**: `utils.WaitSignal()` blocks on SIGTERM/SIGINT.

**Sets**: `sets.NewInt()` / `sets.NewString()` — generic set types with Add/Has/Remove.

**Syncx**: `syncx.NewMap[K, V]()` — generic wrapper around `sync.Map`.

## Anti-Patterns

- **Never** add a `.go` file without a corresponding `_test.go`
- **Never** use `panic` — return errors
- **Never** import from `internal/` packages
- **Always** use `require`/`assert` from testify in tests

## Commands

```bash
go test ./pkg/utils/...
go test -cover ./pkg/utils/...
go tool task test:coverage   # Full coverage report
```
