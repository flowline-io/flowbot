# Agent Note: Tests describe behavior, not correctness

Status: implemented

## Problem

Tests named or asserted as "correct" freeze an accidental snapshot of today's implementation. Changing intended behavior then looks like a test failure to be preserved, so PRs either fight the suite or leave obsolete assertions in place. Package-level tests and fakes also get treated as enough coverage for user-visible assembled paths, so the example stack and BDD/eval transcripts drift.

## Decision

Tests describe behavior, not correctness. When intended behavior changes, the tests change in the same PR, and the PR explains why the old behavior is no longer the contract. Package tests and mocks do not substitute for assembled or product-surface coverage.

Which layer, the example-stack wiring canary, and eval fixture homes live in [docs/testing/README.md](../../../../docs/testing/README.md). Conflict split: [agents-md-conflict-resolution](../process/2026-08-13-agents-md-conflict-resolution.md).

## Alternatives considered

- **Treat a red test as proof the change is wrong.** Rejected: that makes the suite the product spec even when the spec should move.
- **Allow mocks and package tests to stand in for BDD/example coverage.** Rejected: they never exercise registration, HTTP, events, and wiring the way the example stack does.
- **Require a snapshot file format like a keyless CLI transcript.** Rejected for now: Flowbot's assembled transcript is the example stack plus BDD/eval assertions, not a separate replay harness.

## Consequences

- Test names and `It` blocks state the observable outcome (`returns 404 when the hub is missing`), not `works correctly`.
- A behavior change that updates tests without a PR explanation is incomplete.
- Example packages change when the **assembly contract** changes, not when an unrelated product surface changes.

## Verification

Owning docs: [docs/testing/README.md](../../../../docs/testing/README.md). Unit tests name scenarios, not `correct`. Wiring-contract changes update `tests/specs/example_spec_test.go`; product surfaces update their own `tests/specs/` file. There is no separate snapshot replay harness.
