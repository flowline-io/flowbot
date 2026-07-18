# Capability Guide

Capability layer that decouples modules from providers. Each provider-backed capability lives in `pkg/capability/<provider>/`, defines a `Service` interface, and registers via `capability.Register(Spec)`.

## Structure

```
capability/
├── invoke.go, register.go, ability.go (domain types), params, page, cursor
├── operations.go, ops_compat.go   # IsMutation + legacy Op* aliases
├── eventsource.go, polling.go, pool.go, pollstate.go, queries.go, webhook_handler.go, …
├── example/                       # Reference implementation
│   ├── service.go, ops.go, register.go
│   ├── adapter.go, webhook.go, poller.go
│   └── *_test.go
├── karakeep/, miniflux/, kanboard/, trilium/, memos/, fireflyiii/, transmission/, nocodb/, gitea/, github/
├── notify/, agent/                # Internal capabilities (no provider)
└── conformance/
```

## Key Patterns

### Register

```go
func Register(app string, svc Service) error {
    return capability.Register(capability.Spec{
        Type: hub.CapKarakeep, App: app, Description: "...", Instance: svc,
        Ops: []capability.OpDef{
            {Name: OpList, Description: "List", Scopes: []string{auth.ScopeServiceKarakeepRead}, Handler: invokeList(svc)},
            {Name: OpCreate, Mutation: true, Handler: invokeCreate(svc)},
        },
    })
}
```

### Invoke

```go
capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
```

### Rules

- Modules never import `pkg/providers/*` — use `capability.Invoke`.
- Adapters never call hub/pipeline/emit DataEvent directly.
- CapType == provider ID for provider-backed capabilities.
- Domain event names (`bookmark.created`) stay stable; set `DataEvent.Capability` to the provider ID.
- Reference `pkg/capability/example/` for new capabilities.
