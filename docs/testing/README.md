# Testing

When to write which tests, and what they are allowed to claim. Mechanics live in [tdd-specs.md](tdd-specs.md) (table-driven unit tests) and [bdd-specs.md](bdd-specs.md) (Ginkgo acceptance). Standing orders in root [AGENTS.md](../../AGENTS.md) link here; do not restate this file there.

Rationale: [tests describe behavior](../../.agents/notes/implemented/testing/2026-08-13-tests-describe-behavior.md).

## Which layer

| Change | Required |
| --- | --- |
| Library / pure logic in a Go package | Table-driven unit tests ([tdd-specs.md](tdd-specs.md)) |
| New module, or cross-boundary HTTP / event / auth / pipeline behavior of that product surface | Unit tests **and** the owning BDD spec under `tests/specs/` ([bdd-specs.md](bdd-specs.md)) |
| Wiring-contract change (how provider / capability / module register, `Invoke`, and expose HTTP / events) | The [assembled example](#assembled-example) in the same change |
| Agent engine behavior (tools, loop, eval scoring) | Fixtures under `pkg/agent/eval/testdata/` ([pkg/agent/AGENTS.md](../../pkg/agent/AGENTS.md)) |
| Agent product policy (permission, DCG, chatagent safety) | `internal/server/chatagent/eval/testdata/safety/` and `tests/specs/agent_spec_test.go` ([chatagent AGENTS.md](../../internal/server/chatagent/AGENTS.md)) |
| Docs / AGENTS / comment-only | No tests |

Without Docker: always run unit tests (`go tool task test` or package-scoped `go test`). Do not run or claim `test:specs` — state that BDD was skipped.

`httptest` mocks and fake clients are the unit-test layer. They do not replace the owning BDD spec, and they do not replace the example transcript when the assembly contract changes.

## Tests describe behavior, not correctness

Name tests and `It` blocks after the observable outcome (`returns 404 when the hub is missing`), not after correctness (`works`, `correct`, `success path is valid`).

When intended behavior changes, change the tests in the same PR and explain why the old behavior is no longer the contract. Do not preserve assertions that encode the behavior being replaced.

## Assembled example

The example stack is the keyless **wiring canary**, not a dump of every product behavior:

- `pkg/providers/example/`
- `pkg/capability/example/`
- `internal/modules/example/`
- BDD: `tests/specs/example_spec_test.go` against the suite `App` (the registered module, not a stub route that reimplements the handler)

That BDD path is the keyless assembled transcript: no third-party credentials, and no separate snapshot replay harness.

Update that stack when the **assembly contract** changes: registration, `capability.Invoke` wiring, or the example HTTP / event shape. Product behavior of hub, web, karakeep, notify, and other surfaces is recorded in that surface's owning spec under `tests/specs/`.

A path-only move with unchanged behavior does not require a BDD update ([pkg/agent/AGENTS.md](../../pkg/agent/AGENTS.md)).
