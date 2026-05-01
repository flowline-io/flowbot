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

### Architecture Violations

- **Never** import `pkg/providers/*` directly from `internal/modules/*` — use `ability.Invoke`
- **Never** call provider clients directly in modules (e.g. `karakeep.GetClient()`)
- **Never** access local filesystem paths (e.g. `/home/<user>/homelab/apps/<app>/data`) from modules
- **Never** call hub or pipeline from inside a provider
- **Never** emit `DataEvent` from inside a provider
- **Never** return provider-private types from an adapter
- **Never** write cross-service logic in cron handlers — use Pipeline
- **Never** write cross-service logic in event handlers — use Pipeline
- **Never** mount routes under `/service/hub/*` — use `/hub/*` for management plane
- **Never** allow arbitrary shell execution in Hub lifecycle
- **Never** access paths outside `apps_dir` in Hub
- **Never** enable exec/update/pull by default
- **Never** hardcode provider names in workflow definitions
- **Never** hardcode provider names in pipeline definitions
- **Never** use app name instead of capability name for `/service` routes
- **Never** return 500 for all errors — use appropriate status codes
- **Never** return 400 for all errors — use appropriate status codes
- **Never** leak provider raw errors to HTTP layer — wrap with adapter standard errors
- **Never** expose provider pagination internals to API callers
- **Never** expose cursor/offset/page internal structure in API responses
- **Never** use Redis Stream as sole long-term event store — persist to MySQL `data_events` table
- **Never** discard Pipeline execution history — write to database
- **Never** skip delivery record on webhook triggers — save delivery log
- **Never** store API tokens in plaintext
- **Never** skip audit log on Hub lifecycle operations
- **Never** skip idempotency checks in Pipeline steps

## Testing

- Each component has a `*_test.go` counterpart.
- Use table-driven tests with `require`/`assert`.
- Mock external dependencies.
