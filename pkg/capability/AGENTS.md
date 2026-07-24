# Capability Guide

Decouples modules from providers. Provider-backed caps live in `pkg/capability/<provider>/`, register via `capability.Register(Spec)`.

## Entry points

- Core: `invoke.go`, `register.go`, ops/eventsource/polling helpers
- Reference: `example/` (`service.go`, `adapter.go`, `register.go`)
- Provider caps: `karakeep/`, `miniflux/`, …; multi-provider: `devops/`; internal (no provider): `notify/`, `agent/`, `clip/`

```go
capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
```

## Boundaries

- Modules never import `pkg/providers/*` — use `capability.Invoke`
- Adapters never call hub/pipeline/emit DataEvent
- CapType == provider ID for provider-backed caps
- **Exception:** `devops` (`hub.CapDevops`) aggregates beszel/uptimekuma/traefik/grafana/wakapi/dozzle/netalertx (sole multi-provider CapType≠provider). Ops use underscores (`beszel_list_systems`)
- Domain event names stay stable; set `DataEvent.Capability` to provider ID
- New caps: follow `pkg/capability/example/`
