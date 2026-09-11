# Agent Note: Taskfile cleanup

Status: implemented

## Problem

`taskfile.yaml` carried duplicate LOC tooling (`cloc` external binary vs `scc` via `go tool`), sequential wiring for independent checks (`lint` sub-linters, `build:all`, `check`), a wrong `lint:js` description, and section comments that only restated prefixes. Docs advertised `task air` and `task leak` with no tasks, configs, or module entries in the repo.

## Decision

- Keep `scc` only; drop the `cloc` task.
- Parallelize order-independent work with Task `deps`: `lint` (`lint:go`, `lint:testify`, `lint:action`, `lint:js`), `build:all` (native targets; `build:cli:linux` stays opt-in), and `check` (`lint`, `secure`, `gosec`).
- Extract `lint:go` for revive; fix `lint:js` description; ensure `cloc/` exists before writing scc output.
- Remove `air` / `leak` from README and developer-guide instead of restoring tooling.

## Alternatives considered

- Restore `air` + gitleaks (`leak`) to match docs — rejected; no `.air.toml` / gitleaks config and no need to add tooling for doc drift.
- Keep both `cloc` and `scc` — rejected; `default` already uses `scc` via `go tool`, and an external `cloc` binary is an extra install surface.
- Include `build:cli:linux` in `build:all` — rejected; cross-compile remains an explicit sandbox-inject path.

## Consequences

- Callers of `task cloc` must use `task scc`.
- Docs that listed live-reload or gitleaks tasks no longer do.

## Verification

- `go tool task --list-all` has no `cloc`, `air`, or `leak`; CI entrypoints (`lint`, `build`, `build:cli`, `build:gateway`, `build:agent`, `test:specs:ci`, `test:race:ci`, `test:race:nightly`, `agent:eval`) remain.
- README and `docs/developer-guide/README.md` match the surviving task names.
