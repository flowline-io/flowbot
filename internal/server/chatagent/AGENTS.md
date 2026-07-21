# Chatagent (Product Orchestration) Guide

`internal/server/chatagent` is Flowbot's **chat assistant product layer**. It binds the core agent engine (`pkg/agent`) to:

- REST endpoints under `/chatagent/*` (`internal/server/chatagent_http*.go`)
- Web UI pages under `/service/web/agents/*` (module `internal/modules/web`)
- Platform chat sinks (e.g. Slack streaming) via `internal/server/chatagent_handler.go`
- Flowbot persistence via `internal/store`
- Scheduled tasks and delivery context

It must not grow into a second agent engine.

## Key boundaries

- **Allowed dependencies**: `pkg/agent/*` primitives (harness, tools, permission evaluation, sessions), `internal/store/*`, `pkg/views/*` (via handlers).
- **Forbidden**: importing `internal/store` from `pkg/agent` or leaking store-specific types into engine APIs.

## Entry points and routes

- **REST routes**: registered in `internal/server/chatagent_http.go`
  - Primary turn streaming: `POST /chatagent/sessions/:id/messages` (SSE)
  - Observer overlays: `GET /chatagent/sessions/:id/events` (SSE subset)
- **Web observer SSE**: `internal/modules/web/chatagent_web_stream.go`
- **Platform DM**: `internal/server/chatagent_handler.go`
- **Service layer**: `internal/server/chatagent/service.go`

### `/events` filter

Both REST and Web `/events` endpoints forward only approval overlay events:

- `confirm`, `confirm_resolved`, `canceled`, `mode_change`

The shared predicate is `chatagent.IsObserverStreamEvent` in `internal/server/chatagent/protocol.go`.

## Frontend integration

Chatagent UI JavaScript is split into multiple files under `public/js/chatagent-*.js` and uses:

- `window.FlowbotChatAgent` namespace only
- script load order (see `pkg/views/pages/agents.templ`)

Do not re-introduce a monolithic single-file implementation.

Script load order (defer, in order):

1. `chatagent-util.js`
2. `chatagent-sse.js`
3. `chatagent-markdown.js`
4. `chatagent-context.js`
5. `chatagent-approval.js`
6. `chatagent-todos.js`
7. `chatagent-thread.js`
8. `chatagent-chat.js` (boot: composer/thread init)

Pages: `pkg/views/pages/agents.templ`, `agent_session_detail.templ` (approval-only panels may load a subset; thread pages load all).

## Testing

- Unit tests:

```bash
go test ./internal/server/chatagent -count=1
```

- Full suite (includes JS lint via `oxlint`):

```bash
go tool task format
go tool task lint
go tool task test
```

- Specs (BDD, requires Docker/testcontainers in some environments):

```bash
go tool task test:specs
```

## Common refactor safety rules

- Keep HTTP handlers thin; business rules belong in the `chatagent` package.
- Any protocol changes must update `docs/agent/chatagent-feature-checklist.md` and tests first.
- Prefer additive APIs over moving code across package boundaries.

