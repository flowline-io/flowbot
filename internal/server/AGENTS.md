# Server Package

HTTP server (Fiber v3) with fx DI, routing, and protocol handlers.

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

- Never block in handlers — long work in goroutines
- Map `types.Err*` in `error.go`; use `protocol.NewFailedResponse` / `NewSuccessResponse`
- Validate inputs before processing

## Routing

- `/service/{module}/*`, `/hub/*`, `/chatagent/*`, `/static/*` (webassets), `/platform/{platform}` (Slack, Tailchat)
- Also: `/oauth/:provider/:flag`, `/form`, `/agent`, `/metrics`, `/livez`, `/readyz`, `/swagger/*` (`-tags swagger`)

## Testing

```bash
go test ./internal/server/...
```
