# Server Package

HTTP server (Fiber v3) with fx DI, routing, and protocol handlers. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md).

## Entry points

- Bootstrap: `fx.go`, `server.go`, `router.go`, `http.go`
- Lifecycle: `platform.go` (WS), `module.go` (module Init), `database.go`, `event.go`
- Chatagent: `chatagent_bootstrap.go`, `chatagent_http*.go`, `chatagent_handler.go`, package `chatagent/`
- Also: `pipeline.go`, `workflow.go`, `webhook.go`, `hub.go`, `notify.go`, `providers.go`, `readyz.go`, `metrics_auth.go`

Look at the package directory for the full file set; prefer hot-path names above over a 1:1 tree.

## Boundaries / wiring

- **Provide**: constructors in `fx.go` (e.g. `slack.NewDriver`)
- **Invoke**: `handleModules`, `handlePlatform`, OAuth (`providers.go`), notify (`notify.go`); new modules via `fx.Invoke` in `internal/modules/fx.go`
- Tailchat: constructed in controller (not `fx.Provide`). Discord package exists but is **not** wired into the server graph yet.

## Non-obvious rules

- Request-response HTTP handlers must not block; move long work off the handler goroutine.
- SSE / streaming handlers write for the connection lifetime. Do not detach the stream onto a goroutine that outlives the client writer. Chatagent SSE: [chatagent/AGENTS.md](chatagent/AGENTS.md).
- Map `types.Err*` in `error.go`; use `protocol.NewFailedResponse` / `NewSuccessResponse`
- Validate inputs before processing
- Events: DataEvent → PostgreSQL `data_events` (+ event outbox) → Redis Stream → pipeline handler → `pipeline_runs`
- Hub lifecycle operations (start / stop / restart / pull / update) write audit via `writeLifecycleAudit` in `hub.go`. Do not add an unaudited lifecycle path.

## Routing

- `/service/{module}/*`, `/hub/*`, `/chatagent/*`, `/static/*` (webassets), `/platform/{platform}` (Slack, Tailchat)
- Also: `/oauth/:provider/:flag`, `/form`, `/agent`, `/metrics`, `/livez`, `/readyz`, `/swagger/*` (`-tags swagger`)

## Testing

Which layer: [docs/testing/README.md](../../docs/testing/README.md).

```bash
go test ./internal/server/...
```

pkg vs internal gate: [pkg-boundaries.md](../../docs/architecture/pkg-boundaries.md) (`pkg_deps_test.go`).
