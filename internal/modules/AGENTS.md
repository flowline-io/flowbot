# Module Guide

Interaction modules implement command, form, webhook, webservice, cron, page, and event entry points.

## Structure

Each module follows a consistent pattern:

```text
modules/<name>/
├── module.go          # Module state + module.Handler implementation
├── command.go          # Slash/chat commands (optional)
├── form.go             # Interactive forms (optional)
├── cron.go             # Scheduled tasks (optional)
├── event.go            # Legacy module events (optional; prefer DataEvent + Pipeline for new cross-service work)
├── webhook.go          # HTTP webhooks (optional)
├── webservice.go       # HTTP handlers (optional)
├── page.go             # UI pages (optional)
├── instruct.go         # LLM instructions (optional)
├── setting.go          # Module settings (optional)
├── collect.go          # Data collectors (optional)
├── *_test.go           # Tests for each component
└── static/             # Static assets (optional)
```

## Registration

New code should use `pkg/module`. The legacy `pkg/chatbot` package is kept only as a compatibility layer.

## Rules

- Modules are interaction entry points, not provider clients.
- Do not import `pkg/providers/*` from `internal/modules/*`.
- Do not place cross-service orchestration in cron or event handlers; use Pipeline.
- New capability-oriented modules should call `ability.Invoke`.
- Capability webservice routes must use `/service/{capability}` names.
- Hub management routes must be mounted under `/hub/*`, not `/service/hub/*`.

## Testing

- Each component has a `*_test.go` counterpart.
- Use table-driven tests with `require`/`assert`.
- Mock external dependencies.
