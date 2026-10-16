# Agent Note: MCP tool contract

Status: implemented

## Problem

External agents could read only a text blob from `/mcp`. A rate limit, a bad argument, and an internal fault looked the same, so the agent could not decide whether to retry, change parameters, or ask a person. Annotation booleans also marked every write as destructive, and list results had no stable place for a request id.

## Decision

Successful `structuredContent` is the existing result plus a sibling `trace`. JSON arrays are wrapped as `items` plus `trace` only for `pipeline.list`, `workflow.list`, `function.list`, and `hub.apps`. `get_run` without `run_id` returns `{runs, trace}`; a matching `run_id` returns that run object plus `trace`. Failed calls set `isError` and return only `error` and `trace`. Text content is that same JSON. `outputSchema` is a `oneOf` of the success shape and the error envelope for pipeline, workflow, and function `list` / `get` / `run` / `get_run`, `hub.apps`, `hub.health`, and capability operations whose names contain `list`. The success branch rejects a tool-error object (`error` with `code`, `message`, `retryable`, and `hint`). A domain string field named `error` still matches success.

`error` always has `code`, `message`, `retryable`, and `hint`. `message` uses `types.ClientSafeDomainMessage`, except plain errors and `ErrInternal`, which are always `internal error`. `retry_after_ms`, `field`, `job_id`, and `confirmation_required` appear only when set. `types.Error` carries optional `Field`, `RetryAfter`, `JobID`, and `Confirmation`. v1.0 does not fill those optionals at call sites. `ToToolError` lives in `pkg/toolerr`. `internal/server/mcp` re-exports it. It stays out of `pkg/mcp` because `pkg/types` imports `pkg/config`, and `pkg/config` imports `pkg/mcp`. The CLI uses the same function; see [CLI agent output](../feature/2026-10-17-cli-agent-output.md).

`trace.request_id` is the Fiber `X-Request-ID`. `trace.trace_id` is the current W3C span id when the span is valid. `correlation_id` is declared on the schema and omitted until a write emits a `DataEvent`.

`capability.Classify` feeds MCP annotations only. `IsMutation` stays operation-name based for cache and pipelines. Destructive names are `delete`, `delete_*`, `kv_delete`, `archive`, and `remove`. `stop` and `run` are writes. Idempotent hints cover non-destructive `update`, `update_*`, `set_content`, `kv_set`, `attach_tags`, `detach_tags`, `mark_*`, `star_*`, and `unstar_*`. `openWorldHint` is false for `hub`, `pipeline`, `workflow`, and `example`. Titles reuse `{group}: {Description}`.

`data_events` stores `correlation_id`, `causation_id`, and `trace_id` as empty strings until a caller sets them. `correlation_id` is indexed. `ScopeLineageRead` (`lineage:read`) is listed in `AllScopes`.

Contract tests run from `.github/workflows/mcp-contract.yml` via `go tool task test:mcp-contract`, without the Testing workflow `paths-ignore`.

## Alternatives considered

- **One envelope with `ok` / `data` / `error` for every call.** Rejected: existing object results already have their own fields, and wrapping them would change every successful payload.
- **Put the error only in text and keep `structuredContent` for success.** Rejected: clients that validate `outputSchema` would treat a normal `rate_limited` as a schema failure, and text-only clients would not see `retryable`.
- **Emit null for empty `trace` and error fields.** Rejected: agents would have to treat null and missing as the same, and v1.0 would publish keys it never fills.
- **Change `IsMutation` so capability overrides affect cache and audit.** Rejected: `run` is a write for every current caller, and cache behavior should not follow annotation tiers.
- **Lock the future `truncated` / `returned` page shape in v1.0.** Rejected: handlers return `PageInfo` (`limit`, `has_more`, `next_cursor`) or bare arrays. The schema describes those values.
- **Add the contract job inside `testing.yml`.** Rejected: `paths-ignore` on the workflow trigger applies to every job in that file.
- **Put `ToToolError` in `pkg/mcp`.** Rejected: that import cycle is `pkg/types` → `pkg/config` → `pkg/mcp`. It lives in `pkg/toolerr` instead.

## Consequences

- Clients that only read text now parse JSON on both success and failure.
- `pipeline.list`, `workflow.list`, `function.list`, and `hub.apps` return an object with `items` instead of a bare array. `get_run` without `run_id` returns `runs`.
- Omitted `destructiveHint` still defaults to true in the MCP SDK, so write tools set the pointer explicitly. Omitted `openWorldHint` defaults to true, so closed groups set false explicitly.
- `lineage:read` can be minted before a lineage read API exists. It does not grant other scopes.
- Rate-limit responses omit `retry_after_ms` until a caller sets `types.Error.RetryAfter`. `job_id` and `confirmation_required` stay omitted until a caller sets `JobID` or `Confirmation`.

## Verification

- `go tool task test:mcp-contract`
- BDD: `tests/specs/mcp_spec_test.go` (`tools/call` error envelope, `hub.apps` items, `hub.health` object). Those calls send `Mcp-Protocol-Version: 2026-07-28` matching `_meta`, plus `Mcp-Method` and `Mcp-Name`. The streamable handler returns 400 when that protocol is declared and the headers are absent.
- `go test ./pkg/types/protocol/ ./pkg/mcp/ ./internal/server/mcp/ ./pkg/capability/ -count=1 -run "TestNewFailedResponse|TestToToolError|TestOutputSchemaContract|TestClassify|TestContractAnnotations"`
- `go test ./internal/store/ -count=1 -run TestDataEventTraceColumns`
- User guide: [mcp.md](../../../docs/user-guide/mcp.md)
