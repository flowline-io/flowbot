---
name: flowbot-dev-loop
description: >-
  Default Flowbot agent development loop — read nested AGENTS/example, explore-plan-implement,
  table-driven tests, optional BDD, then fix, lint, and test. Use when starting feature work, refactors,
  or any multi-step coding task in this repo; when the user asks for the default workflow;
  or when attaching @flowbot-dev-loop.
disable-model-invocation: true
---

# Flowbot Dev Loop

Checkable steps for coding in this repository. Complements root `AGENTS.md` hard gates and the `tdd` / `grill-me` / `split-to-prs` skills — do not duplicate their full text.

## Steps

1. **Orient** — Read the nearest nested `AGENTS.md`. When touching providers, capabilities, or modules, also open the matching `example/` package.
2. **Explore → Plan → Implement** — For unclear boundaries or multi-package changes, use Plan mode or `/grill-me` before coding. Prefer small diffs in existing files.
3. **Tests** — Logic changes: table-driven unit tests first (see `docs/testing/tdd-specs.md`). New modules or cross-boundary behavior: add or update BDD under `tests/specs/` (`docs/testing/bdd-specs.md`). Docs/AGENTS-only: skip tests.
4. **Verify** — Before finishing: `go tool task fix`, then `go tool task lint`, then relevant `go tool task test` (or package-scoped `go test`). Without Docker: do not run or claim `test:specs`; say BDD was skipped.
5. **Commit** — NEVER git commit unless the user asked.

## Cursor Cloud

If the environment has no systemd (Cursor Cloud): read `docs/developer-guide/cursor-cloud.md` before starting services or relying on local DSN/Redis assumptions.
