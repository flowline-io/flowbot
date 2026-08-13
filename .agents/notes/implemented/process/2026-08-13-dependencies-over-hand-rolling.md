# Agent Note: Prefer maintained dependencies over hand-rolling

Status: implemented

## Problem

Hand-rolled SSE parsers, retry/backoff loops, glob matchers, and similar infrastructure accrete from an unstated "avoid new dependencies" reflex. Each clone is code this repo tests, documents, reviews, and debugs, without the ecosystem's edge-case fixes. Nothing in standing orders stated a dependency policy, so agents inferred "don't add deps" from existing owned packages such as `pkg/backoff`.

## Decision

Introducing an external dependency is a legitimate simplification, not a policy exception. When a well-maintained Go module (or a stdlib API at our language floor) covers a hand-rolled surface, replacing the hand-rolled code is the preferred direction.

The bar for a new dependency:

- **Net deletion.** The dependency replaces real owned code (implementation + dedicated tests + docs), not hypothetical future code. A dependency that only adds capability is a feature decision, not a simplification. Wrapping a library without deleting the owned surface fails the bar.
- **Health.** Actively maintained, widely used, sensible transitive footprint. A tiny unmaintained module trades our code for abandoned code.
- **Fit at the boundary.** The module's semantics cover the actual contract; residual semantics still hand-rolled around it count against the swap.
- **Not a settled seam.** Decisions already recorded in implemented Agent Notes (sonic instead of `encoding/json`; LLM retry only before first stream delta in `pkg/agent/llm`; chatagent SSE framing and event protocol) are not reopened by this policy; a swap that collapses a recorded design must beat that rationale.

Hand-rolling SSE, retry/backoff, glob, or equivalent protocol/parsing infrastructure is the rejected default. A proposal to swap owned infrastructure for a dependency is a `proposed/simplification` Agent Note: candidate module, deletable surface, residual semantics, and supply-chain cost.

## Alternatives considered

- **Keep an implicit no-new-deps culture.** Rejected: it was never a recorded decision, and its cost is owned protocol and parsing code that duplicates battle-tested libraries.
- **A hard allowlist of approved modules.** Rejected: the set is still small; a per-PR evidence bar (net deletion, health, fit) plus review keeps judgment where the context is.
- **Vendor every new dependency.** Rejected: vendoring is for packages we must patch or pin against upstream churn. Ordinary Go modules with `go.mod` pinning are the default.

## Consequences

- Agents surveying for simplifications treat "replace hand-rolled X with module Y" as in-scope when the swap genuinely shrinks owned code.
- The module graph may grow; pin versions in `go.mod` and run `go tool task tidy`.
- Root `AGENTS.md` carries the one-line rule; this note owns the rationale and the bar.

## Verification

A dependency swap is recorded as a `proposed/simplification` Agent Note that names the deletable owned surface. Settled seams (sonic; LLM retry until first stream delta; chatagent SSE protocol) stay recorded here and in the owning package `AGENTS.md`. Root `AGENTS.md` links this note and does not restate the bar.
