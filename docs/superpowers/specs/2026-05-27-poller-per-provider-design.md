# Poller Per-Provider Relocation

## Motivation

Currently poller implementations live at the capability package level (`pkg/ability/example/poller.go`, `pkg/ability/note/poller.go`). Polling behavior — cursor strategy, diff key field names, content hash logic — is inherently provider-specific. A Trilium poller uses `"noteId"` as diff key; a future Joplin adapter would use different fields. The capability-level poller creates a false abstraction that can't accommodate provider diversity.

Additionally, poller registration is hardcoded in `internal/server/pipeline.go` via direct adapter imports, while webhooks are correctly registered in `internal/modules/hub/module.go` `Bootstrap()`. This inconsistency creates unnecessary coupling between the server package and adapter packages.

## Design Overview

Each provider adapter owns its `PollingResource` implementation, following the same pattern as `WebhookConverter`. Poller files move from capability level to adapter directories alongside `adapter.go` and `webhook.go`. Registration consolidates in `hub/module.go` `Bootstrap()` where webhooks are already registered.

## Architecture

```
Layers (unchanged):
  Module (hub/module.go Bootstrap())
    → EventSourceManager.RegisterPolling(poller)
      → poller implements PollingResource
        → poller calls Service.ListRawEvents()
          → adapter (pkg/ability/<cap>/<backend>/adapter.go)
            → provider client (pkg/providers/<backend>/)

File locations (changed):
  Before:
    pkg/ability/example/poller.go         → capability-level poller
    pkg/ability/note/poller.go            → capability-level poller

  After:
    pkg/ability/example/example/poller.go → adapter owns poller (same dir as adapter.go, webhook.go)
    pkg/ability/note/trilium/poller.go    → adapter owns poller (same dir as adapter.go)
```

The `PollingResource` interface (`pkg/ability/eventsource.go`) and `EventSourceManager` need no changes. The `Service` interface keeps `ListRawEvents()` — pollers still call through the same adapter.

Registration moves from `pipeline.go` to `hub/module.go` `Bootstrap()`, mirroring webhook registration:

```go
func (moduleHandler) Bootstrap() error {
    mgr := ability.GetEventSourceManager()
    // Webhooks (existing)
    mgr.RegisterWebhook(karakeepAdapter.NewWebhook())
    mgr.RegisterWebhook(minifluxAdapter.NewWebhook())
    // ...

    // Pollers (new)
    mgr.RegisterPolling(karakeepAdapter.NewPoller(karakeepSvc))
    mgr.RegisterPolling(triliumAdapter.NewPoller(triliumSvc))
    // ...
}
```

## Files & Changes

### New files

| File                                         | Purpose                                                       |
| -------------------------------------------- | ------------------------------------------------------------- |
| `pkg/ability/example/example/poller.go`      | Relocated ExamplePoller (was `pkg/ability/example/poller.go`) |
| `pkg/ability/example/example/poller_test.go` | Relocated poller tests                                        |
| `pkg/ability/note/trilium/poller.go`         | Relocated NotePoller (was `pkg/ability/note/poller.go`)       |
| `pkg/ability/note/trilium/poller_test.go`    | Relocated poller tests                                        |

### Deleted files

| File                                 | Purpose                                               |
| ------------------------------------ | ----------------------------------------------------- |
| `pkg/ability/example/poller.go`      | Moved to `pkg/ability/example/example/poller.go`      |
| `pkg/ability/example/poller_test.go` | Moved to `pkg/ability/example/example/poller_test.go` |
| `pkg/ability/note/poller.go`         | Moved to `pkg/ability/note/trilium/poller.go`         |
| `pkg/ability/note/poller_test.go`    | Moved to `pkg/ability/note/trilium/poller_test.go`    |

### Modified files

| File                                     | Change                                                                                                                                                                     |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/ability/example/example/poller.go`  | Package `example` → adapter package; `NewExamplePoller()` → `NewPoller()`; compile-time check imports `ability.PollingResource`                                            |
| `pkg/ability/note/trilium/poller.go`     | Package `note` → `trilium`; `NewNotePoller()` → `NewPoller()`; imports `note.Service` interface                                                                            |
| `pkg/ability/example/example/adapter.go` | Remove `NewExamplePoller()` factory — poller is now self-contained in same package                                                                                         |
| `pkg/ability/note/trilium/adapter.go`    | Remove `NewNotePoller()` factory — poller is now self-contained in same package                                                                                            |
| `internal/server/pipeline.go`            | Remove example adapter import; remove `RegisterWebhook(exampleAdapter.NewExampleWebhook())` with TODO comment; remove `RegisterPolling(exampleAdapter.NewExamplePoller())` |
| `internal/modules/hub/module.go`         | Add adapter imports; add poller registration calls in `Bootstrap()`                                                                                                        |
| `pkg/ability/AGENTS.md`                  | Update PollingResource section to reflect per-provider location and `NewPoller()` convention                                                                               |

## Key Design Decisions

### Constructor naming: `NewPoller()`

Follows the `NewWebhook()` convention. Already namespaced by import (`karakeepAdapter.NewPoller()`), so prefix is unnecessary inside the adapter package.

### Service creation: inline in constructor

`NewPoller()` creates its own service instance internally via the adapter's `New()`. This matches the `NewWebhook()` pattern and avoids cross-package service reference passing. For testing, a `NewPollerWithService(svc Service)` variant accepts an injected service.

### Service interface: unchanged

`ListRawEvents()` stays on the capability `Service` interface. Providers that don't support polling simply return `nil, "", nil` from that method. This keeps the poller decoupled from the provider client — it still goes through the adapter layer.

### Registration ordering

The `EventSourceManager` must be created (in `initPipeline`) before modules call `Bootstrap()`. This ordering already exists in `internal/server/fx.go`. The poller registration follows the same startup sequence as webhook registration.

## Anti-Patterns Avoided

- Poller does not call provider client directly — still goes through `Service.ListRawEvents()`
- Registration does not live in `pipeline.go` — follows the Bootstrap convention
- Each adapter directory owns its complete provider surface (adapter + webhook + poller)

## Testing

- Existing poller tests relocate with the source files, updating imports and package names
- Conformance: `var _ ability.PollingResource = (*NotePoller)(nil)` compile-time check stays
- No new conformance suite needed since `PollingResource` is already covered by `eventsource_test.go`
