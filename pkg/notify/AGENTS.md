# Notify Guide

Multi-channel notification gateway: rules → templates → channel `Notifyer` dispatch.

Call path: callers → `notify.GatewaySend()` (or optional `pkg/capability/notify`) → rules/template engines → registered providers (`slack`, `ntfy`, `pushover`, `messagepusher`).

## Entry points

- Core: `notify.go` (`Register`, `Send`, `GatewaySend`), `types.go` (`Notifyer`)
- Engines: `template/`, `rules/` (templates/rules load from PostgreSQL, not YAML)
- Channels: `pkg/notify/<name>/provider.go`; wire via `fx.Invoke` in `internal/server/notify.go`

```go
type Notifyer interface {
    Protocol() string
    Templates() []string
    Send(tokens types.KV, message Message) error
}
```

## Add a provider (checklist)

1. `pkg/notify/<name>/provider.go` — package-level `plugin`, implement `Notifyer`
2. `Register()` → `notify.Register(ID, &handler)`; URI templates with `{placeholders}`
3. `Send` → internal `doSend(tokens, msg, client, baseURL)` for httptest injection
4. `fx.Invoke` in `internal/server/notify.go`
5. Table-driven `provider_test.go` with `httptest.NewServer`

Conventions: `resty.dev/v3`; wrap errors with protocol prefix; credentials only from `tokens types.KV`.

## Non-obvious rules

- Never import `pkg/providers/*` from a notify provider
- Never emit DataEvent / call hub/pipeline from `Send()`
- Never block event handlers — `Send()` stays non-blocking
- Distinguish connection vs downstream API errors (do not map all to 500)

## Testing

```bash
go test ./pkg/notify/...
```

BDD: `tests/specs/notify_spec_test.go`.
