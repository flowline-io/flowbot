# Agent Note: Kanboard health getMe poisons shared JSON-RPC client

Status: implemented

## Problem

Pipeline steps such as `get_task` failed with `unexpected HTTP status 403 Forbidden` even when the same credentials and endpoint succeeded from inside the Flowbot container via `wget`. Retries kept failing until process restart. The Kanboard Application API user `jsonrpc` cannot call `getMe` (HTTP 403). Capability health probes on `/service/web/healthz/capabilities` invoke `kanboard.health`, which called `getMe`. `jrpc2`'s HTTP channel treats any non-200 as a channel error and permanently stops the shared client, so every later Kanboard RPC reused that 403.

## Decision

1. Probe health with `getVersion`, which Application API and User API both allow.
2. Wrap the Kanboard HTTP transport so non-OK responses become JSON-RPC error bodies with HTTP 200 (preserving request `id` and a truncated response body in the message). That keeps the shared `jrpc2` client alive when Kanboard or a proxy returns 401/403 for a single method.

## Alternatives considered

- **Keep `getMe` and document User API credentials only** — breaks the common `jsonrpc` + API token setup used in production.
- **Recreate the client after channel stop** — still races health probes against in-flight calls and hides the root wrong probe method.
- **Skip Kanboard in healthz capability probes** — leaves `getMe` as a footgun for CLI/`capability.Invoke` health and does not fix other non-200 poison paths.

## Verification

- `go test ./pkg/providers/kanboard/ ./pkg/capability/kanboard/`
- `TestKanboard_HTTPForbiddenDoesNotPoisonClient` asserts `getMe` HTTP 403 then `getTask` still succeeds on the same client.
- After deploy: open healthz (capability poll), then run `get_task` / retry `complete-bookmark-task` without restarting Flowbot.
