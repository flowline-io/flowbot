# Types Package

Shared types: rulesets, message payloads, protocol, KV, models.

## Entry points

- Core: `types.go`, `kv.go`, `msg.go`, `errors.go`, `event.go`, …
- Protocol: `protocol/` (Driver/Adapter/Action, messages, errors)
- Rulesets: `ruleset/{command,form,webservice}/`
- DTOs: `model/`; also `audit/`, stats helpers (`pipeline_stats`, `token_usage_*`, `run_latency_stats`)

Look at the package directory for the full file list.

## Boundaries

- Prefer `KV` over raw `map[string]any` for structured access (`map[string]any` OK in `capability.Invoke` params / protocol / generated code)
- Never define new message types outside this package
- New message types must implement `MsgPayload.Convert()`

## Testing

```bash
go test ./pkg/types/...
```
