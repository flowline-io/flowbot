# Module Guide

Interaction entry points: command, form, webhook, webservice, cron, page, event. Not provider clients. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md).

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

- Provider wiring belongs in `pkg/capability/<provider>/adapter.go`
- Routes: `/service/{module}/*` for module business APIs; hub **management** APIs under `/hub/*` — never put hub management under `/service/hub/*` (hub module business routes may still live under `/service/hub`)

## Testing

Which layer: [docs/testing/README.md](../../docs/testing/README.md). Owning BDD specs live under `tests/specs/`.
