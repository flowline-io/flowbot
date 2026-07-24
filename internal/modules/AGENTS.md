# Module Guide

Interaction entry points: command, form, webhook, webservice, cron, page, event. Not provider clients.

## Entry points

```text
modules/<name>/
├── module.go       # moduleHandler + module.Base, Register(), Init(), Rules(), Webservice()
├── command.go / form.go / webhook.go / webservice.go
├── *_test.go
└── utils.go
```

`Register()` wired via `fx.Invoke` in `internal/modules/fx.go` → `modules.Modules`. Reference: `internal/modules/example/`.

## Boundaries

- Do not import `pkg/providers/*` — use `capability.Invoke`
- Provider wiring belongs in `pkg/capability/<provider>/adapter.go`
- Routes: `/service/{module}/*` for module business APIs; hub **management** APIs under `/hub/*` — never put hub management under `/service/hub/*` (hub module business routes may still live under `/service/hub`)
- Cross-service orchestration in Pipeline, not cron/event handlers

## Testing

Table-driven unit tests; BDD under `tests/specs/`.
