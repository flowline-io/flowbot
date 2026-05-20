# Store Context Propagation Design

**Status**: Draft
**Date**: 2026-05-20

## Problem

All database queries in the store layer use `context.Background()`, creating a hard boundary where upstream tracing, deadlines, cancellation are lost:

- 119 occurrences of `context.Background()` in store layer (86 in Postgres adapter alone)
- Zero methods in `store.Adapter` (92 methods) accept `context.Context`
- Specialized stores (`RunStore`, `WorkflowRunStore`, `AuditStore`, `EventStore`, `HubStore`) also all use `context.Background()` internally
- Upper layers (HTTP, pipeline, workflow, `ability.Invoke`) already propagate `context.Context` properly — it is dropped at the store boundary

This causes:

- OpenTelemetry traces break at the store boundary (no parent span propagation to database queries)
- Cancellation from HTTP request timeouts never reaches database calls
- No deadline propagation for long-running store operations from pipeline/workflow contexts

## Goal

Add `ctx context.Context` as the first parameter to every data-access method across all store interfaces, pass through the caller's context, and remove internal `context.Background()` creation.

## Design

### 1. Interface Changes

#### `store.Adapter` (86 data-access methods get `ctx`)

All CRUD methods get `ctx context.Context` as the first parameter:

```go
// Before
UserCreate(user *model.User) error
UserGet(uid types.Uid) (*model.User, error)
DataSet(uid types.Uid, topic string, key string, val any) error

// After
UserCreate(ctx context.Context, user *model.User) error
UserGet(ctx context.Context, uid types.Uid) (*model.User, error)
DataSet(ctx context.Context, uid types.Uid, topic string, key string, val any) error
```

**Skipped** (lifecycle/infrastructure, keep as-is):
`Open`, `Close`, `IsOpen`, `GetName`, `Stats`, `GetDB`, `Migrate`

#### Specialized Stores

Same pattern — `ctx context.Context` as first parameter to all data-access methods:

- `RunStore` (11 methods in `pkg/pipeline/engine.go`)
- `WorkflowRunStore` (19 methods in `pkg/workflow/`)
- `AuditStore.Write()` (`internal/store/audit_store.go`)
- `EventStore.AppendDataEvent`, `EventStore.AppendEventOutbox` (`internal/store/event_store.go`)
- `HubStore.SaveHomelabApps()` (`internal/store/hub_store.go`)

### 2. Call Site Context Sources

Every call site already has a context available. Changes are purely mechanical — add `ctx` as first argument:

| Layer | Context source | Example |
|---|---|---|
| HTTP handlers | `fiber.Ctx.Context()` | `store.Database.UserGet(c.Context(), uid)` |
| Module command/cron | Already receives `ctx *types.Context` | Extract `ctx.Context` or pass directly |
| Pipeline engine | Already has `ctx context.Context` | Pass through |
| Workflow runner | Already has `ctx context.Context` | Pass through |
| Core packages (event, notify, media) | Invoked from modules, have ctx available | Pass through |
| `Migrate()` | Standalone function, not in interface | Keep `context.Background()` |

**Fire-and-forget writes**: Audit entries, counter increments, and writes that must not be cancelled during request cleanup will use `context.Background()` intentionally. Identified per-call-site during implementation.

### 3. Implementation Order

Bottom-up, 4 commits within one PR for reviewability:

1. **Specialized stores** — Add `ctx` to `RunStore`, `WorkflowRunStore`, `AuditStore`, `EventStore`, `HubStore` interfaces + implementations + call sites (~6 files, smallest blast radius).

2. **Postgres adapter implementation** — Add `ctx` params to all 86 data-access methods in `internal/store/postgres/adapter.go`. Replace `ctx := context.Background()` with passed `ctx`. Compiles against old interface until step 3.

3. **Adapter interface + all call sites** — Add `ctx` to `store.Adapter` interface in `internal/store/store.go`. Update all 97 call sites across 23 files. Mechanical change: `Foo(args...)` → `Foo(ctx, args...)`.

4. **Verify** — Run `go build`, `go vet`, `revive`, full test suite.

### 4. What is NOT Changed

- `internal/store/ent/gen/` — Generated code, untouched
- Service architecture — No introduction of dependency injection, no splitting of `Adapter` into sub-interfaces
- `store.Database` global variable — Remains as-is; callers just pass `ctx` through it
- No wrapping or adapter layers — Direct parameter addition

## Risks and Mitigation

| Risk | Mitigation |
|---|---|
| Missed call sites | Compiler catches every one — no runtime discovery |
| Large PR | 4 logical commits within one PR, each independently reviewable |
| Fire-and-forget writes cancelled | `context.Background()` kept at identified call sites |
| Merge conflicts | Single atomic change; proceed when no active store PRs are in flight |

## Verification

- `go build ./...` — Compiles all binaries
- `go vet ./...` — Passes vet checks
- `revive ./...` — Passes lint
- `go tool task test` — All unit tests pass
- `go tool task test:specs` — BDD acceptance tests pass
