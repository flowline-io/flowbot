# Agent Note: Light agent, heavy ops

Status: implemented

## Problem

After cutting parallel agent workers, dormant plugin hosts, unused providers, and empty modules, nothing recorded the product stance those removals enforce. Without that stance, a later change can reintroduce a second heavy execution plane beside the chat agent, or invent a second parameter-binding hash for approval that drifts from write idempotency.

## Decision

Flowbot keeps a **light agent** and puts **heavy ops** on the capability plane.

The chat agent (and its sandbox) is the conversational and coding loop. Durable side effects — deploy, ledger, kanban, pipeline or workflow runs — go through `capability.Invoke`, Pipelines, Workflows, Hub, MCP, REST, and the CLI. Those paths own audit, delivery, and idempotency. Flowbot does not grow a second pull-worker, CapGateway, headless coding CLI, out-of-tree plugin host, or unused provider client to carry those ops.

That stance is what the 2026-10-21 simplifications already shipped:

- [Remove local CLI gateway and headless agent](../simplification/2026-10-21-remove-local-cli-gateway-and-headless-agent.md)
- [Remove wasm and gRPC plugins](../simplification/2026-10-21-remove-wasm-grpc-plugin.md)
- [Remove instruct and ghost configs](../simplification/2026-10-21-remove-instruct-and-ghost-configs.md)
- [Remove uncalled providers](../simplification/2026-10-21-remove-uncalled-providers.md)
- [Merge lineage module into web](../simplification/2026-10-21-merge-lineage-module-into-web.md)

**Shared parameter hash (suggestions 18 and 22).** Approval binding (suggestion 18: bind a confirmation to the approved operation's business parameters) and write idempotency (suggestion 22: [write idempotency keys](../feature/2026-10-21-write-idempotency-key.md)) use the same parameter-hash primitive: RFC 8785 canonical JSON then SHA-256, as `pkg/idempotency.Hash` / `HashJSON`. Empty and `{}` share a hash; JSON null does not. Token, operation name, and the caller-supplied key or confirmation id stay outside the hash. Shipping the approval/`confirmation_id` surface can wait for v1.1; this note only freezes the shared-hash convention so a later approval design does not invent a second digest.

## Alternatives considered

- **Treat the agent as the heavy execution plane** (gateway jobs, headless worker, CapGateway). Rejected: that duplicates auth, leases, and coding tools beside the chat-agent sandbox, which is what the 2026-10-21 gateway/agent removal closed.
- **Separate digests for approval binding and idempotency.** Rejected: two canonicalizations for the same business payload will disagree on edge cases (empty vs `{}`, number forms) and force callers to reason about which hash a confirmation or replay checked.
- **Ship `confirmation_id` with the idempotency work.** Rejected during the suggestion-22 grill: the hash function is the shared primitive; confirmation storage, one-time use, and the approval UX are their own design and can land in v1.1.

## Consequences

- New heavy side effects extend capabilities, pipelines, or workflows — not a parallel agent worker stack.
- A future approval-binding design reuses `pkg/idempotency` hashing; it does not add a second canonical JSON path.
- The write-idempotency note already states that `confirmation_id` is out of that change and may call the same hash later; this process note is the home for that cross-feature convention.

## Verification

- The five 2026-10-21 simplification notes above remain the shipped removals that keep the agent light.
- `pkg/idempotency.Hash` / `HashJSON` are the only parameter-hash helpers for write idempotency; a future approval binder imports them rather than reimplementing RFC 8785.
- Tree search for a second parameter-hash package under `pkg/` stays empty aside from `pkg/idempotency`.
