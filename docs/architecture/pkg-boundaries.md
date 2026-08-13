# pkg vs internal Boundaries

Flowbot treats `pkg/` as shared application libraries that must stay independent of `internal/` implementation details. Product orchestration and ORM bindings live under `internal/`.

This is an **architectural** rule (enforced by `internal/server/pkg_deps_test.go`), not Go's `internal/` visibility rule (same-module imports are legal).

## Rules

1. `pkg/*` must not import `internal/*`, except packages on the migration allowlist in `internal/server/pkg_deps_test.go`.
2. `pkg` public APIs and templates must not expose `*gen.*` or `internal/store` facades — use `pkg/types` / `pkg/types/model` and inject interfaces from `internal/server`.
3. Domain enums used outside the store layer live in `pkg/types` (e.g. Instruct*, FormState, PipelineState, WorkflowRunState, ResourceRef); `ent/schema` may type-alias them — do not redefine parallel constants.
4. `internal/store` continues to return `*gen.*` to product layers; mapping to `pkg/types` happens in adapters / web services — not inside store facades.

Positive examples already clean: `pkg/capability`, `pkg/agent` (orchestration in `internal/server/chatagent`).

## Migration waves

| Wave | Scope | Target shape |
|------|-------|--------------|
| **L1** (done) | `pkg/types`, `pkg/providers`, `pkg/media` | Injected stores; no `internal` imports; architecture gate + AGENTS |
| **L3** (done) | `pkg/views` | Stay in `pkg`; templates only consume `pkg/types/model`; mapping in `internal/modules/web`; drop `ent/gen` and `internal/server/chatagent` imports |
| **L2** (done) | `pkg/route`, `pkg/module`, `pkg/event`, `pkg/notify` | Contracts stay in `pkg`; inject store interfaces from `internal/server` |
| **L4** (done) | `pkg/pipeline`, `pkg/workflow` | Engine contracts stay in `pkg`; DTO interfaces + `internal/store` adapters; no `*gen.*` in pkg |

After each wave, remove the cleaned packages from the architecture-test allowlist. The allowlist is now empty.

## Gate

```bash
go test ./internal/server -run TestPkgMustNotImportInternal
```

Standing order (one line, links here): root [AGENTS.md](../../AGENTS.md).
