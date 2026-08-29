# Capability Guide

Decouples modules from providers. Provider-backed caps live in `pkg/capability/<provider>/`, register via `capability.Register(Spec)`. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md).

## Entry points

- Framework: `invoke.go`, `register.go`, ops/eventsource/polling helpers
- Reference: `example/` (`service.go`, `adapter.go`, `register.go`)
- Provider caps: `karakeep/`, `miniflux/`, …; multi-provider: `devops/`; internal multi-op: `core/`
- Shared exec for sandbox run_*: `pkg/exec` (used by `core` and agent coding tools)

```go
capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
capability.Invoke(ctx, hub.CapCore, core.OpNotifySend, map[string]any{...})
```

## Boundaries

- CapType == provider ID for provider-backed caps
- **Exceptions** (CapType ≠ single provider ID):
  - `devops` (`hub.CapDevops`) aggregates beszel/uptimekuma/traefik/grafana/wakapi/dozzle/netalertx/scanopy
  - `core` (`hub.CapCore`) aggregates notify/clip/agent plus runtime primitives (`http_request`, `run_code`, `run_terminal`, `kv_*`)
  - `functions` (`hub.CapFunctions`) named FaaS invoke/get/health
  - `life` (`hub.CapLife`) solo Life gamification AI
  - `gateway` (`hub.CapGateway`) local CLI gateway workers
- Domain event names are a recorded consumer contract (pipelines, webhooks). They stay stable under the 0.x no-shim stance. Set `DataEvent.Capability` to provider ID.
- Pagination: limit + opaque cursor; hide provider pagination internals in the adapter
- New caps: follow `pkg/capability/example/`
- `pkg/capability/core` injects `Persister` / `KVStore` / `Runner` / `ExecProvider` from `internal/server` ([pkg-boundaries.md](../../docs/architecture/pkg-boundaries.md))
- `pkg/exec` must not import `pkg/capability`
