# Module Guide

Interaction modules implement command, form, webhook, webservice, cron, page, and event entry points.

## Structure

```text
modules/<name>/
├── module.go       # moduleHandler + module.Base, Register(), Init(), Rules(), Webservice()
├── command.go      # Slash/chat commands
├── form.go         # Interactive forms
├── webhook.go      # HTTP webhooks
├── webservice.go   # HTTP handlers
├── *_test.go       # Tests (TDD: table-driven)
└── utils.go        # Helper functions
```

Module `Register()` functions are wired via `fx.Invoke` in `internal/modules/fx.go` (pulled into the server app through `modules.Modules`).

## Reference Implementation

- When creating or modifying a module, reference `internal/modules/example/` for file structure, naming, and code style.
- `module.go`: `moduleHandler` struct embedding `module.Base`, `Register()` → `module.Register(Name, &handler)`, `Init(jsonconf) error` with `configType{Enabled bool}`, `Rules() []any`, `Webservice(app)`
- `webservice.go`: `webservice.Rule` definitions, route handlers call `capability.Invoke()`
- `webhook.go`: Webhook route rule; hub modules may register webhooks directly in `Bootstrap()` via `EventSourceManager.RegisterWebhook()`

## Rules

- Modules are interaction entry points, not provider clients
- Do not import `pkg/providers/*` from `internal/modules/*` — use `capability.Invoke` or go through the adapter layer
- New capability modules call `capability.Invoke`
- Provider wiring happens inside the capability adapter (`pkg/capability/<provider>/adapter.go`), not in the module
- Webservice routes: `/service/{module}/*` (see `pkg/route`), management: `/hub/*`
- Cross-service orchestration in Pipeline, not cron/event handlers

## Testing

- Prefer a `*_test.go` counterpart for each non-trivial component
- Table-driven tests with `require`/`assert`
- BDD integration tests live under `tests/specs/` (Ginkgo v2 + Gomega)
- Mock external dependencies
