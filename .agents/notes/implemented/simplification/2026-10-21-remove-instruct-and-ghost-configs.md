# Agent Note: Remove instruct and ghost configs

Status: implemented

## Problem

The desktop instruct path (table, pull/ack actions, `instruct:push` handler, `InstructMsg`) stored commands for a host agent and acknowledged them. Nothing in the product created those rows except tests, and the push handler returned immediately. Pocket, Safeline, Wallos, and Telegram were configuration only: vendor stubs in the reference YAML, or a `platform.telegram.enabled` flag with no driver.

## Decision

Flowbot does not persist or serve instructs. The desktop agent protocol keeps online, offline, and message.

Removed with instruct: the Ent `instruct` schema and store methods, pull/ack actions, `InstructMsg` and its platform card conversion, and the `instruct:push` subscriber. Deployments drop the leftover table with the statements in [deployment.md](../../../../docs/developer-guide/deployment.md#database-schema-upgrades).

Removed with the ghosts: `platform.telegram`, and the `vendors.pocket`, `vendors.safeline`, and `vendors.wallos` blocks in the reference config. Notify channel examples name registered providers (`slack`, `ntfy`, `pushover`, `message-pusher`, `inapp`).

The config loader ignores a leftover `platform.telegram` block. Extra keys under `vendors` stay unread.

## Alternatives considered

- Keep pull/ack as no-ops — rejected. Callers would still send a protocol that stores nothing.
- Leave the vendor stubs for a later provider — rejected. Pre-release prefers one config surface, and these names have no client in the tree.
- Drop Discord with Telegram — rejected. Discord has a platform package; it is unwired, which is a different gap.

## Consequences

- Host agents report online, offline, and message only.
- Existing PostgreSQL instances keep the `instruct` table until ops runs `DROP TABLE`.
- Operators drop `platform.telegram` and the three vendor blocks from `flowbot.yaml` when they want the file to match the template. The loader does not fail on those keys.

## Verification

- `go tool task ent`
- `go generate ./pkg/config/`
- `go test ./pkg/types/... ./pkg/config/... ./pkg/event/... ./internal/server/ ./internal/platforms/ ./internal/store/`
- `go tool task lint`
