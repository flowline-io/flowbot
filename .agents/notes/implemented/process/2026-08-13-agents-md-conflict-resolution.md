# Agent Note: Resolve AGENTS.md rule conflicts

Status: implemented

## Problem

Subtree `AGENTS.md` files restated root standing orders and, in five places, contradicted them: domain event names vs 0.x rename-freely; example-stack coverage vs per-module BDD; agent eval fixture homes; HTTP handler non-blocking vs SSE; web `store.Database` vs store connection-only APIs.

## Decision

Each conflict has one home and a subtree exception or narrowed scope:

- **Domain event names** stay stable as a recorded consumer contract. Home: [pkg/capability/AGENTS.md](../../../../pkg/capability/AGENTS.md). The 0.x no-shim stance does not rename them.
- **Assembled example** covers wiring-contract changes only. Product behavior is recorded in the owning `tests/specs/` file. Home: [docs/testing/README.md](../../../../docs/testing/README.md).
- **Agent transcripts** split: engine fixtures in `pkg/agent/eval/testdata/`; product policy (permission / DCG) in `internal/server/chatagent/eval/testdata/safety/` plus `tests/specs/agent_spec_test.go`.
- **HTTP**: request-response handlers must not block; SSE handlers write for the connection lifetime. Home: [internal/server/AGENTS.md](../../../../internal/server/AGENTS.md). Chatagent SSE protocol is a settled seam.
- **Web data access**: `XxxStoreFromDB()` / `NewXxxStore`, never `store.Database.<BusinessMethod>`. Home: [internal/store/AGENTS.md](../../../../internal/store/AGENTS.md).
- **pkg vs internal** lives in [pkg-boundaries.md](../../../../docs/architecture/pkg-boundaries.md); root `AGENTS.md` links there.
- **LLM retry until first stream delta** is a settled seam in [pkg/agent/AGENTS.md](../../../../pkg/agent/AGENTS.md).
- **AuthContext call paths** (REST, CLI, Chat, Webhook, Cron, Pipeline, Workflow) live in [pkg/auth/AGENTS.md](../../../../pkg/auth/AGENTS.md).
- **Hub lifecycle audit** lives in [internal/server/AGENTS.md](../../../../internal/server/AGENTS.md).

Subtree files link root standing orders and keep only tree-specific orders.

## Alternatives considered

- **Keep both copies and add "unless" footnotes.** Rejected: the copies were the conflict.
- **Make example specs the only assembled transcript for every product surface.** Rejected: hub / web / notify already have owning BDD files; forcing example would duplicate behavior in the wrong home.

## Consequences

- Agents read the nested `AGENTS.md` for tree-specific orders and follow links for repo-wide rules.
- Wiring-contract changes still update `pkg/providers/example`, `pkg/capability/example`, `internal/modules/example`, and `tests/specs/example_spec_test.go`.

## Verification

Homes in this Decision match the owning files: event names in [pkg/capability/AGENTS.md](../../../../pkg/capability/AGENTS.md); layers in [docs/testing/README.md](../../../../docs/testing/README.md); SSE write rules in [internal/server/AGENTS.md](../../../../internal/server/AGENTS.md); store call sites in [internal/store/AGENTS.md](../../../../internal/store/AGENTS.md); pkg vs internal in [pkg-boundaries.md](../../../../docs/architecture/pkg-boundaries.md). Root `AGENTS.md` does not restate those bodies.
