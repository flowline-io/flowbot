# Agent Note: Agent sandbox image packaging

Status: implemented

## Problem

`flowbot-agent-sandbox` ships a full official Go tarball, apt documentation, and build-time npm/`GOCACHE` trees in the same layers as the toolchains Cloud Agents and named FaaS actually need. That unpacked weight hits every `:latest` / `base` pull. The published toolchain contract (Ubuntu 24.04, git, Go, NodeSource Node 22, Python, oxfmt/oxlint, dcg, `build-essential`, optional Playwright Chromium) is not the same as "keep every file the installer unpacked."

## Decision

Packaging of [`deployments/agent-sandbox/Dockerfile`](../../../../deployments/agent-sandbox/Dockerfile) deletes non-contract files in the same `RUN` that created them. The toolchain list and NodeSource install path do not change.

- After unpacking the official Go tarball: keep `GOROOT/src` and the compiler (Go 1.21+ compiles the standard library from source; FaaS offline `go run` needs that). Delete `GOROOT/test`, `doc`, `api`, `lib/wasm`, and `testdata` directories.
- After apt (including NodeSource `nodejs`, and again after Playwright `install-deps`): delete `/usr/share/man`, `/usr/share/doc`, `/usr/share/info`, apt lists and archives. Keep locale data and `tzdata`.
- After `npm install -g` / `npx`: `npm cache clean --force`, remove `/root/.npm` or `/home/agent/.npm`, and delete `/tmp` contents in that same `RUN`. Playwright Chromium under `~/.cache/ms-playwright` stays.
- After the FaaS `go run` smoke: delete `/home/agent/.cache/go-build` (`GOCACHE`) in that same `RUN`. The first runtime `go run` is a cold stdlib compile.

CI [`.github/workflows/docker-agent-sandbox.yml`](../../../../.github/workflows/docker-agent-sandbox.yml) prints `docker image inspect` Size after pull. There is no failing size ceiling until a measured baseline exists. Functional smoke is unchanged.

User-facing contract stays in [Agent Sandbox](../../../../docs/agent/agent-sandbox.md) (official Go tarball, NodeSource, Playwright Chromium). The strip list is not part of that contract.

Hermetic Go / `GOTOOLCHAIN=local` remains [agent sandbox FaaS Go toolchain](./2026-08-14-agent-sandbox-faas-go-toolchain.md).

## Alternatives considered

- Change the published toolchain contract (drop a language, Alpine, FaaS-only image, drop `build-essential`) — rejected; size was a packaging problem. A FaaS-only split was already rejected in the FaaS toolchain note.
- Delete all of `GOROOT/src` — rejected; it breaks offline `go run` / `go test` of user code that imports the standard library.
- Keep smoke-test `GOCACHE` to warm first `go run` — rejected; the cache is stdlib-only, does not help real Agent workspaces, and gives back much of the tarball saving.
- Strip locales down to `C.UTF-8` — rejected; `LANG=C.UTF-8` plus Playwright/Python make locale deletion a larger runtime risk than the remaining megabytes.
- Switch Node from NodeSource to an official binary tarball — rejected; the contract table names NodeSource.
- Fail CI above a megabyte ceiling — rejected; there is no measured baseline, and a ceiling would become a second contract.

## Consequences

- First `go run` / `go test` in a new container recompiles the standard library from `GOROOT/src`.
- `go test` of the standard library itself may fail without `testdata`; Cloud Agent and FaaS contracts test workspace or stdlib-only `main.go`, not `GOROOT`.
- Image Size in CI logs is observational. A future ceiling needs its own note and a number taken from those logs.

## Verification

- Dockerfile: Go strip, apt doc removal, npm cache, `/tmp`, and `GOCACHE` deletion share the `RUN` that produced those files; Playwright `install-deps` repeats man/doc/info deletion; Playwright browser install does not delete `~/.cache/ms-playwright`.
- `.github/workflows/docker-agent-sandbox.yml` smoke: existing toolchain / offline `go run` / Playwright version checks, plus `image_size_bytes=` from `docker image inspect` (no size `exit 1`).
- Contract home: [docs/agent/agent-sandbox.md](../../../../docs/agent/agent-sandbox.md).
