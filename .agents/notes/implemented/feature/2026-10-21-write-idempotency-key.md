# Agent Note: Write idempotency keys

Status: implemented

## Problem

The agent error contract says when a call may be retried, and it does not promise that the retry will skip work that already happened. On a homelab that repeats a deploy, a ledger transaction, or a kanban card.

## Decision

Optional idempotency keys cover non-read MCP tools, automate write routes except `POST /service/automate/functions/call`, hub capability write routes, and the CLI flag `--idempotency-key`. MCP sends `arguments.idempotency_key`. REST and the CLI send `Idempotency-Key`.

The claim is a PostgreSQL row keyed by access-token id, operation, and key. There is no TTL and no Redis lock. The unique constraint is the claim. A stopped process moves leftover `in_progress` rows to `unknown` at startup.

The business payload is hashed with RFC 8785 canonical JSON and SHA-256. Empty and `{}` share a hash. JSON null does not. Numbers at or beyond 1e21 use the ECMAScript exponent form. The same key with a different hash is `invalid_argument` and does not execute. The same hash replays a stored success, returns `unavailable` with `retry_after_ms` 1000 while the first call is running, or returns `idempotency_unknown` (HTTP 409, exit 5, `retryable: false`) when the outcome is unknown or the success body was larger than `mcp.max_output_bytes`. Replay does not call the handler, so it does not emit a second data event. The row stores the first correlation id, including non-mutation writes such as stop and run, and REST writes. Replay reuses that id. The first response trace does not add `correlation_id`; a replay sets `trace.replayed` and `trace.correlation_id`. An existing row is answered after scope checks and before the MCP tools/call rate limit. The first claim passes auth, scope, and argument validation before that limit, then inserts. Argument validation does not leave a row. Handler errors after insert mark the row `unknown`. A keyed write whose access token cannot be resolved does not run the handler.

`POST /service/automate/functions/call` keeps its function-run key. `confirmation_id` is not part of this change; it can call the same hash function later.

## Alternatives considered

- Redis with a 12–24 hour TTL. A restart or expiry would run the write again, which breaks the at-most-once promise.
- Release the key on every failure, matching n8n's public API. A timeout after the provider accepted the call would deploy, book, or create a card twice.
- Reuse `conflict` for an unknown outcome. That code is `retryable: true` and tells the caller to read and write again.
- Put the key in MCP `_meta`. The model can set tool arguments; it usually cannot set `_meta`.
- Scope the key by user instead of token. Another token could read the stored result.
- Require the key. Current CLI, web, and unkeyed callers would all fail, and a caller can still mint a new key per attempt.

## Consequences

Callers that mint one key before the first write and reuse it on retries do not repeat the side effect. Callers that omit the key, or that change the key, still execute again. A crash after the provider accepts the call leaves the key `unknown`; the caller lists or gets before using a new key. Success bodies over the output cap are not stored, and a replay of that key is `idempotency_unknown`. REST replay is the `Idempotency-Replayed: true` header. CLI JSON adds `replayed: true`. MCP sets `trace.replayed`.

## Verification

`go test` for `pkg/idempotency`, `internal/store` idempotency, `pkg/toolerr`, `pkg/types/protocol`, `cmd/cli/utils`, and `internal/server/mcp`. `go tool task lint`. The MCP BDD spec covers a missing required argument leaving the key reusable. `go tool task test:specs` was not run.
