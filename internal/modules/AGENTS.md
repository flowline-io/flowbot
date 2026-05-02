# Module Guide

Interaction modules implement command, form, webhook, webservice, cron, page, and event entry points.

## Structure

```text
modules/<name>/
├── module.go       # State + module.Handler implementation
├── command.go      # Slash/chat commands
├── form.go         # Interactive forms
├── cron.go         # Scheduled tasks
├── event.go        # Legacy events (prefer DataEvent + Pipeline for cross-service)
├── webhook.go      # HTTP webhooks
├── webservice.go   # HTTP handlers
├── page.go         # UI pages
├── *_test.go       # Tests for each component
└── static/         # Static assets
```

## Rules

- Modules are interaction entry points, not provider clients
- Do not import `pkg/providers/*` from `internal/modules/*`
- New capability modules call `ability.Invoke`
- Webservice routes: `/service/{capability}`, management: `/hub/*`
- Cross-service orchestration in Pipeline, not cron/event handlers

## Testing

- Each component has `*_test.go` counterpart
- Table-driven tests with `require`/`assert`
- Mock external dependencies
