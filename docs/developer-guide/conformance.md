# Ability Adapter Conformance Tests

Standard compliance suite for ability adapters. Any new provider adapter must pass this suite to guarantee pagination and error handling conform to Flowbot standards.

Source: `pkg/ability/conformance/`

## Overview

Each capability (bookmark, archive, reader, kanban) defines a `Service` interface in `pkg/ability/<capability>/interface.go`. Provider adapters implement this interface, translating provider-specific types into ability domain types. The conformance suite verifies every adapter correctly implements:

- Pagination structure (`ListResult[T]`, `PageInfo`, opaque cursors)
- Error wrapping (Flowbot sentinel errors via `types.WrapError` / `types.Errorf`)
- Input validation (empty IDs, nil queries, missing required fields)
- Context cancellation (canceled ctx maps to `types.ErrTimeout`)
- Not-implemented semantics (unsupported ops return `types.ErrNotImplemented`)

## Architecture

```
pkg/ability/conformance/
├── conformance.go          # Shared types, context helpers, error assertion helpers
├── pagination.go           # Pagination-specific assertions (cursor round-trip, PageInfo structure)
├── bookmark.go             # RunBookmarkConformance — 33 subtests across 9 operations
├── archive.go              # RunArchiveConformance — 7 subtests across 3 operations
├── reader.go               # RunReaderConformance — 20 subtests across 7 operations
├── kanban.go               # RunKanbanConformance — 27 subtests across 9 operations
├── conformance_test.go     # Self-tests for the conformance framework
└── pagination_test.go      # Self-tests for pagination helpers
```

Each adapter adds a `conformance_test.go` in its package:

```
pkg/ability/bookmark/karakeep/
├── adapter.go              # Adapter implementation
├── adapter_test.go         # Adapter-specific tests (type conversion, edge cases)
└── conformance_test.go     # Calls conformance.RunBookmarkConformance
```

## Design

### Config-Based Factory Pattern

Each capability runner accepts a `Config` struct and a factory function. The factory translates `Config` fields into fake client behavior:

```go
type BookmarkConfig struct {
    ListItems      []*Bookmark    // Items returned by List
    ListNextCursor string         // Next cursor from provider
    ListErr        error          // Error from List
    // ... per-operation fields
}

type BookmarkServiceFactory func(t *testing.T, cfg BookmarkConfig) Service
```

The runner creates subtests, each with a different `Config`. The adapter's `conformance_test.go` implements the factory, mapping `Config` fields to its fake client:

```go
conformance.RunBookmarkConformance(t, func(t *testing.T, cfg conformance.BookmarkConfig) bm.Service {
    c := &fakeClient{
        listResp:  toProviderResponse(cfg),
        listErr:   cfg.ListErr,
        // ...
    }
    a := NewWithClient(c).(*Adapter)
    a.cursorSecret = conformance.CursorSecret
    return a
})
```

This decouples the conformance runner from provider-specific types — the runner only knows about ability domain types.

### Per-Capability Runners

Each runner function defines all test cases for its capability's operations. Test cases follow a consistent pattern:

| Dimension       | Operations Tested | Verification                                           |
| --------------- | ----------------- | ------------------------------------------------------ |
| Success         | All               | Non-nil result, correct item fields                    |
| Pagination      | List, Search      | `PageInfo` present, `HasMore` logic, `Items` never nil |
| Timeout         | All               | `errors.Is(err, types.ErrTimeout)`                     |
| Invalid input   | All               | `errors.Is(err, types.ErrInvalidArgument)`             |
| Provider error  | Mutations         | `errors.Is(err, types.ErrProvider)`                    |
| Not implemented | Per-backend       | `errors.Is(err, types.ErrNotImplemented)`              |

## What the Suite Tests

### Pagination Conformance (every list/search operation)

| Test                | Assertion                                                 |
| ------------------- | --------------------------------------------------------- |
| Non-nil Items       | `result.Items` is `[]*T{}` (empty slice, not nil)         |
| Non-nil Page        | `result.Page` is always present                           |
| Limit pass-through  | Non-zero limit preserved (or normalized by adapter)       |
| HasMore = true      | When provider returns a next cursor                       |
| HasMore = false     | When provider returns no next cursor                      |
| NextCursor encoding | Cursor is HMAC-signed opaque string when HasMore is true  |
| Cursor decoding     | Incoming opaque cursor extracts provider cursor correctly |

### Error Conformance (every operation)

| Sentinel Error             | When Expected                                          |
| -------------------------- | ------------------------------------------------------ |
| `types.ErrTimeout`         | Context is canceled before operation                   |
| `types.ErrProvider`        | Provider client returns an error                       |
| `types.ErrInvalidArgument` | Empty ID, empty URL, nil tags, missing required fields |
| `types.ErrNotFound`        | Entity does not exist (where applicable)               |
| `types.ErrNotImplemented`  | Operation not supported by this backend                |

### Input Validation (every operation)

- Empty/missing IDs return `ErrInvalidArgument`
- Empty URLs return `ErrInvalidArgument`
- Nil query structs use safe defaults (no panic)
- Empty tag slices return `ErrInvalidArgument`

### Context Cancellation (every operation)

- Canceled context before any provider call returns `ErrTimeout`
- Wrap uses `types.WrapError(types.ErrTimeout, "...", ctx.Err())`

## Adding a New Provider

### Step 1: Implement the Service Interface

Create an adapter in `pkg/ability/<capability>/<provider>/` implementing the capability's `Service` interface.

### Step 2: Create a Fake Client

In `adapter_test.go`, define a `fakeClient` struct implementing the adapter's local `client` interface. Each method should have configurable response and error fields.

### Step 3: Write Adapter-Specific Tests

In `adapter_test.go`, test adapter internals: type conversion functions, edge cases, boundary conditions.

### Step 4: Wire Up the Conformance Suite

Create `conformance_test.go`:

```go
package newprovider

import (
    "testing"
    "github.com/flowline-io/flowbot/pkg/ability/conformance"
    bm "github.com/flowline-io/flowbot/pkg/ability/bookmark"
)

func TestConformance(t *testing.T) {
    conformance.RunBookmarkConformance(t, func(t *testing.T, cfg conformance.BookmarkConfig) bm.Service {
        c := &fakeClient{
            listResp:  toProviderListResponse(cfg),
            listErr:   cfg.ListErr,
            // ... map every config field
        }
        a := NewWithClient(c).(*Adapter)
        if cursorAdapter, ok := interface{}(a).(interface{ SetCursorSecret([]byte) }); ok {
            cursorAdapter.SetCursorSecret(conformance.CursorSecret)
        }
        return a
    })
}
```

### Step 5: Run the Tests

```bash
# Run this adapter's conformance only
go test -run TestConformance ./pkg/ability/bookmark/newprovider/

# Run all ability tests
go test ./pkg/ability/...

# Run all tests
go tool task test
```

## Cursor Secrets

Adapters using cursor-based pagination must expose a way to set the cursor secret for testing. The conformance suite provides `conformance.CursorSecret` and `conformance.TestTime()` for deterministic cursor encoding.

If the adapter has a `SetCursorSecret([]byte)` method:

```go
a.SetCursorSecret(conformance.CursorSecret)
```

If the adapter exposes a `now` field for time injection (deterministic cursor expiry):

```go
a.now = conformance.TestTime
```

## Assertion Helpers

The conformance package exports reusable assertion helpers:

| Function                                          | Purpose                                            |
| ------------------------------------------------- | -------------------------------------------------- |
| `RequireListResult[T](t, result, limit, hasMore)` | Verifies `ListResult` structure                    |
| `RequireTimeoutError(t, err)`                     | Asserts `errors.Is(err, types.ErrTimeout)`         |
| `RequireProviderError(t, err)`                    | Asserts `errors.Is(err, types.ErrProvider)`        |
| `RequireInvalidArgError(t, err)`                  | Asserts `errors.Is(err, types.ErrInvalidArgument)` |
| `RequireNotFoundError(t, err)`                    | Asserts `errors.Is(err, types.ErrNotFound)`        |
| `RequireNotImplementedError(t, err)`              | Asserts `errors.Is(err, types.ErrNotImplemented)`  |
| `AssertCursorRoundTrip(t, secret, payload)`       | Verifies cursor encode → decode                    |
| `AssertPageInfoIsComplete(t, page, limit)`        | Verifies all `PageInfo` fields                     |
| `CanceledContext()`                               | Returns an already-canceled `context.Context`      |

## Coverage Matrix

| Adapter    | Capability | Conformance Tests | Adapter-Specific Tests | Total   |
| ---------- | ---------- | ----------------- | ---------------------- | ------- |
| karakeep   | bookmark   | 33                | 2                      | 35      |
| archivebox | archive    | 7                 | 1                      | 8       |
| miniflux   | reader     | 20                | 6                      | 26      |
| kanboard   | kanban     | 27                | 7                      | 34      |
| **Total**  |            | **87**            | **16**                 | **103** |

Plus 12 self-tests for the conformance framework itself.

## Extending the Suite

To add conformance coverage for a new capability:

1. Define `<capability>Config` and `<Capability>ServiceFactory` types in a new `pkg/ability/conformance/<capability>.go`
2. Implement `Run<Capability>Conformance` with subtests covering all operations
3. Wire up existing adapters by adding `conformance_test.go` in each adapter package
4. Add new subtests if the capability has unique semantics
