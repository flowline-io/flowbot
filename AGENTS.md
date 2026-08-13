# Flowbot

Homelab Data Hub & Capability Orchestration Center. Stack: Go 1.26.5+, PostgreSQL, Redis.

## Pre-release stance: foundation over blast radius

Until 1.0, prefer the correct foundation over compatibility shims. Domain event names stay stable ([capability](pkg/capability/AGENTS.md)). Rewrite this section at 1.0. Rationale: [.agents/notes/implemented/process/2026-08-13-pre-release-foundation-over-shims.md](.agents/notes/implemented/process/2026-08-13-pre-release-foundation-over-shims.md).

## Coding guidelines

* Prioritize code correctness and clarity. Speed and efficiency are secondary unless otherwise specified.
* Prefer implementing functionality in existing files unless it is a new logical component. Avoid creating many small files.
* Exported symbols must have godoc. Unexported symbols: no comment by default; comment only to explain non-obvious why.
* Do not write organizational or summary comments that restate the code.
* Do not use emojis.
* Text in English: comments, docs, commit messages, code.
* NEVER git commit unless asked.
* Style that lint covers (imports, naming, JS quotes): follow `go tool task lint` / `revive.toml` / oxlint — do not restate here.

## Decisions and docs

* Non-trivial changes include an Agent Note in the same change. [scope](.agents/notes/README.md)
* One home per fact: state a rule in its owning tier; elsewhere, link. Do not restate the same rule in `AGENTS.md`, architecture docs, and README. [tiers](docs/AGENTS.md)

## Verification

* Before editing a package, read the nearest nested `AGENTS.md` and the matching `example/` when touching providers, capabilities, or modules.
* After modifying code, run `go tool task lint`, then the relevant tests (`go tool task test` or package-scoped `go test`).

## Testing policy

* Tests describe behavior, not correctness. When intended behavior changes, change the tests in the same PR and explain why. Layers, Docker skip, and assembled example: [policy](docs/testing/README.md).

## Dependencies

* Prefer a maintained dependency over hand-rolling SSE, retry/backoff, or glob. Net-deletion bar and settled seams: [policy](.agents/notes/implemented/process/2026-08-13-dependencies-over-hand-rolling.md).

## Boundaries

* pkg vs internal import and API surface: [pkg-boundaries.md](docs/architecture/pkg-boundaries.md). Gate: `go test ./internal/server -run TestPkgMustNotImportInternal`.
* Never import `pkg/providers/*` from `internal/modules/*` — use `capability.Invoke` (do not call provider clients from modules).
* Never call hub / pipeline / emit DataEvent from a provider or capability adapter; never return provider-private types from an adapter.
* Never write database query code outside `internal/store` (domain `*Store` facades in package `store`; `postgres` is connection-only).
* Never edit generated code.
* Never edit `docs/skills/` directly — it is generated. Change `cmd/composer/action/skills/` then run `go tool task skills`.
* Never use `encoding/json` Marshal / Unmarshal — use `github.com/bytedance/sonic` (`json.RawMessage` from stdlib is allowed).
* Never use `panic` outside initialization; never ignore errors.
* Never block in event handlers; never use Redis Stream as the sole event store — persist to PostgreSQL `data_events`.
* Never skip delivery / audit / idempotency records.
* Never write cross-service logic in cron / event handlers — use Pipeline.
* Never hardcode provider names in pipeline / workflow definitions.
* Never return 500/400 for all errors; never leak provider raw errors or pagination internals to the HTTP layer.
* Use `http.NoBody` instead of `nil` in `http.NewRequest` calls.

## References

* Provider example: `pkg/providers/example/`
* Capability example: `pkg/capability/example/`
* Module example: `internal/modules/example/`
* Format: `go tool task format`
* Lint: `revive` (strict, see `revive.toml`)
* Errors: wrap with `%w`; use `types.ErrNotFound` / `ErrForbidden` / `ErrProvider`

## Commands

```bash
go tool task build            # Main server
go tool task lint             # Code lint
go tool task test             # Unit tests
go tool task test:specs       # BDD acceptance tests (requires Docker)
go tool task test:specs:ci    # BDD with retry + JUnit
go tool task ent              # Generate ent code from database
go tool task skills           # Regenerate docs/skills from cmd/composer
```

## Configuration

* Runtime: `flowbot.yaml` (copy from `docs/reference/config.yaml`)
* Build: `taskfile.yaml`
* Lint: `revive.toml`
* CI: `.github/workflows/build.yml`

## See also

* Cursor Cloud / environments without systemd: read [docs/developer-guide/cursor-cloud.md](docs/developer-guide/cursor-cloud.md) first.
* Default agent loop (optional `@` skill): [`.cursor/skills/flowbot-dev-loop/SKILL.md`](.cursor/skills/flowbot-dev-loop/SKILL.md)
* Nested package guides: nearest `AGENTS.md` under `internal/` / `pkg/` / `cmd/`
* Documentation standard (one home per fact): [docs/AGENTS.md](docs/AGENTS.md)
* Agent Notes: [.agents/notes/README.md](.agents/notes/README.md)
* Testing policy: [docs/testing/README.md](docs/testing/README.md)
