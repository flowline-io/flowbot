# Config Startup Validation

## Overview

Add startup configuration validation to fail fast before any subsystem initializes. The config struct undergoes pure field validation followed by dependency reachability checks. Invalid config blocks startup with clear, actionable error messages.

## Motivation

Currently `config.Type` at `pkg/config/config.go:31` has 20+ fields with zero validation. Config is unmarshaled via viper and used as-is. Each subsystem performs ad-hoc checks at init time (e.g., `rdb.NewClient()` checks `addr == ":"` at `pkg/rdb/rdb.go:28`, postgres adapter checks `DSN == ""` at `internal/store/postgres/adapter.go:100`). Invalid config causes obscure runtime errors instead of clear startup failures.

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Accumulate all errors | User fixes everything in one pass, avoids multiple restart loops |
| Skip disabled subsystems | `Enabled: false` means don't validate — avoids requiring config for unused features |
| Separate reachability from field validation | Field validation is fast/pure; reachability requires network I/O with timeouts |
| Methods on `config.Type` | Idiomatic Go, co-located with struct, minimal new types |
| No validation on hot-reload | Invalid hot-reload logs a warning; doesn't crash a running server |

## Architecture

### New file: `pkg/config/validate.go`

Two methods on `config.Type`:

```go
// Validate performs pure field validation (no I/O). Accumulates all errors.
func (t *Type) Validate() error

// ReachabilityCheck attempts DB and Redis connections with short timeouts.
// Only meaningful after Validate() passes.
func (t *Type) ReachabilityCheck(ctx context.Context) error
```

**Multi-error type:**

```go
type ValidationErrors []error

func (ve ValidationErrors) Error() string {
    var b strings.Builder
    for i, e := range ve {
        if i > 0 { b.WriteByte('\n') }
        b.WriteString(e.Error())
    }
    return b.String()
}
```

**Error format:**
```
<field.path>: <description>. Fix: <yaml-key>
```
Example:
```
redis.host: must not be empty. Fix: set redis.host in flowbot.yaml
redis.port: must be between 1 and 65535, got 0. Fix: set redis.port in flowbot.yaml
```

### Call site: `pkg/config/config.go` — `NewConfig()`

After `Load()` succeeds, before the fx lifecycle hook:

```go
if err := App.Validate(); err != nil {
    return nil, fmt.Errorf("config validation failed:\n%w", err)
}
if err := App.ReachabilityCheck(context.Background()); err != nil {
    return nil, fmt.Errorf("dependency check failed:\n%w", err)
}
```

### Validation techniques

- **Struct tags** (`validate:"..."`) on sub-struct fields for checks the `go-playground/validator` library handles natively (required, min, max, url, oneof, etc.). Uses existing `validate.Validate` from `pkg/validate/rules.go`.
- **Imperative checks** in `Validate()` for cross-field logic: extracting DSN from `any` map, cross-referencing agent models, duration parsing, host:port parsing.

## Validation Scope

### Always-required fields

| Field | Check | Fix hint |
|-------|-------|----------|
| `redis.host` | non-empty | `set redis.host in flowbot.yaml` |
| `redis.port` | 1—65535 | `set redis.port in flowbot.yaml` |
| `redis.password` | non-empty | `set redis.password in flowbot.yaml` |
| `store_config.use_adapter` | non-empty | `set store_config.use_adapter in flowbot.yaml` |
| `store_config.adapters` | contains key matching `use_adapter` | `set store_config.adapters.<name> in flowbot.yaml` |
| `store_config.adapters.<name>.dsn` | non-empty (extracted from `any` map) | `set store_config.adapters.<name>.dsn in flowbot.yaml` |

### Format validation (always when present)

| Field | Check |
|-------|-------|
| `listen` | valid `host:port` if non-empty |
| `log.level` | one of `debug`, `info`, `warn`, `error`, `fatal`, `panic` |
| `log.rotation.maxSize` | > 0 when rotation block present |
| `log.rotation.maxBackups` | >= 0 when rotation block present |
| `flowbot.url` | valid URL if non-empty |
| `homelab.discovery.probe_timeout` | valid Go duration string if non-empty |
| `capability.event_pool.expiry_duration` | valid Go duration string if non-empty |

### Conditional (only when `.Enabled`)

| Subsystem | Checks |
|-----------|--------|
| `platform.slack` | `app_id`, `client_id`, `client_secret`, `signing_secret` non-empty |
| `platform.discord` | `app_id`, `public_key`, `client_id`, `client_secret`, `bot_token` non-empty |
| `platform.tailchat` | `api_url` is valid URL |
| `tracing` | `endpoint` valid URL, `sample_rate` 0.0—1.0 |
| `profiling` | `server_address` valid URL |
| `models[i]` | `provider` non-empty, `base_url` valid URL |
| `agents[i]` | `name` non-empty, `model` references known `models[*].name` |

### ReachabilityCheck

| Target | Method | Timeout |
|--------|--------|---------|
| PostgreSQL | `sql.Open("pgx", dsn)` + `db.PingContext(ctx)` | 5s |
| Redis | `redis.NewClient(opts)` + `client.Ping(ctx)` | 3s |

### What is NOT validated

- `modules` / `vendors` — raw `any` maps, consumed by individual modules/providers
- `pipelines` — already validated by `pkg/pipeline/loader.go`
- `notify.templates/rules` — validated by `pkg/notify/rules/engine.go`
- `executor` — executor-specific, not always needed
- `search.url_base_map` — free-form map, optional

## Existing Ad-Hoc Checks to Remove

### `pkg/rdb/rdb.go:28-30`

Remove the `addr == ":" || password == ""` check. `NewClient` receives pre-validated config. Keep the `Ping` at line 35 (connection establishment, not config validation).

### `internal/store/postgres/adapter.go:100-102`

Remove the `conf.DSN == ""` check. Already guaranteed by `Validate()`.

### Kept as-is

- `pkg/config/config.go:631-640` — `ApiPath` default-setting (normalization, not validation)
- `internal/store/postgres/adapter.go:104-106` — `SqlTimeout` default-setting (normalization)

## Hot-Reload Behavior

`viper.OnConfigChange` at `pkg/config/config.go:647` calls `viper.Unmarshal(&App)`. We add `App.Validate()` after unmarshal. On failure, log a warning and keep the previous config — do not crash a running server:

```go
if err := App.Validate(); err != nil {
    log.Printf("[config] Reloaded config is invalid, keeping previous: %v", err)
}
```

## Testing

### Unit: `pkg/config/validate_test.go`

Table-driven as required by AGENTS.md. At least 3 cases per test function.

**`TestValidate_Required`** — each required field absent → error contains field path + fix hint.

| Case |
|------|
| missing redis host |
| missing redis password |
| redis port zero |
| redis port too high |
| missing store adapter name |
| use_adapter not found in adapters map |
| missing DSN |

**`TestValidate_Format`** — format checks on fields.

| Case |
|------|
| valid log level passes |
| invalid log level fails |
| tracing enabled, missing endpoint |
| tracing enabled, invalid URL |
| sample rate out of range |
| invalid listen address |
| invalid probe timeout duration |
| invalid expiry duration |

**`TestValidate_Conditional`** — Enabled-gated.

| Case |
|------|
| slack enabled, missing app_id |
| slack disabled, missing app_id (no error) |
| discord enabled, missing bot_token |
| discord disabled (no error) |
| tailchat enabled, invalid api_url |
| agent references unknown model |
| model missing provider |
| model invalid base_url |

**`TestValidate_Accumulated`** — multiple errors returned together.

| Case |
|------|
| redis host + DSN both empty → 2 errors |

**`TestValidate_HappyPath`** — fully valid config → nil error.

**`TestReachabilityCheck_*`** — skipped by default (`testing.Short()`), gated by env vars.

## Files Changed

| File | Change |
|------|--------|
| `pkg/config/validate.go` | **New** — `Validate()`, `ReachabilityCheck()`, `ValidationErrors`, helpers |
| `pkg/config/validate_test.go` | **New** — table-driven unit tests |
| `pkg/config/config.go` | Call `Validate()` + `ReachabilityCheck()` in `NewConfig()`; add validation to hot-reload |
| `pkg/rdb/rdb.go` | Remove line 28-30 ad-hoc check |
| `internal/store/postgres/adapter.go` | Remove line 100-102 ad-hoc check |
