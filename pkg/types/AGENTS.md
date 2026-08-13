# Types Package

Shared types: rulesets, message payloads, protocol, KV, models. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md). pkg vs internal: [pkg-boundaries.md](../../docs/architecture/pkg-boundaries.md).

## Entry points

- Core: `types.go`, `kv.go`, `msg.go`, `instruct.go`, `form_state.go`, `run_state.go`, `resource.go`, `errors.go`, `event.go`, …
- Protocol: `protocol/` (Driver/Adapter/Action, messages, errors)
- Rulesets: `ruleset/{command,form,webservice}/`
- DTOs: `model/`; also `audit/`, stats helpers (`pipeline_stats`, `token_usage_*`, `run_latency_stats`)

Look at the package directory for the full file list.

## Boundaries

- Prefer `KV` over raw `map[string]any` for structured access (`map[string]any` OK in `capability.Invoke` params / protocol / generated code)
- Never define new message types outside this package
- New message types must implement `MsgPayload.Convert()`
- Instruct*, FormState, PipelineState, WorkflowRunState, ResourceRef/Edge, PipelineNamePattern/ValidatePipelineName and other shared domain enums/types owned here; store schemas type-alias when needed
- UI/transport DTOs stay under `model/` (including pipeline/workflow run rows and ResourceLink); do not leak ORM entities into this package

## Testing

```bash
go test ./pkg/types/...
```
