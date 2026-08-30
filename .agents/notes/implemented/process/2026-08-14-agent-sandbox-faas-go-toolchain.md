# Agent Note: Agent sandbox FaaS Go toolchain

Status: implemented

## Problem

Named FaaS runs `go run main.go` inside `flowbot-agent-sandbox` with `Network=none` and `GOPROXY=off` (stdlib only). The official Go tarball defaults to `GOTOOLCHAIN=auto`, which can attempt a toolchain download when the module language version and bundled compiler disagree or when auto mode still probes the proxy. Under FaaS isolation that download cannot succeed, so Go functions fail even though the image already ships a Go compiler.

## Decision

The agent sandbox image pins a hermetic Go runtime for FaaS and Cloud Agents:

- Install Go `${GO_VERSION}` (matches root `go.mod`) at `/usr/local/go` with `PATH` and `GOROOT` set
- Image `ENV GOTOOLCHAIN=local` so the bundled toolchain never auto-downloads
- Build-time smoke as `agent`: offline `go run main.go` with a minimal `flowbotfn` / `go 1.26` module
- CI smoke: assert `go env GOTOOLCHAIN=local` and rerun offline `go run` under `--network=none`
- `pkg/exec.mergeGoEnv` also sets `GOTOOLCHAIN=local` (with existing `GOPROXY=off` / `GOSUMDB=off` / `CGO_ENABLED=0`) so FaaS process env matches the image contract even on older images that only have a mismatched Go

Cloud Agents that need a different toolchain can override `GOTOOLCHAIN` when the container has network access.

## Alternatives considered

- Leave `GOTOOLCHAIN=auto` and rely only on version alignment — rejected; FaaS `Network=none` + `GOPROXY=off` has no recovery path if auto tries to fetch
- Separate slim FaaS-only image — rejected for now; one sandbox image already serves chat-agent Exec and FaaS; split cost outweighs benefit
- Bake module cache / prebuilt binaries into the image — rejected; FaaS source is ephemeral per invoke and must stay stdlib-only

## Consequences

- Image Go is hermetic by default; agents that want auto toolchain download must set `GOTOOLCHAIN=auto` (and have network)
- FaaS Go functions remain stdlib-only; third-party imports still fail under `GOPROXY=off`
- Sandbox releases that change `GO_VERSION` must keep the FaaS `go.mod` language line (`go 1.26`) compatible with the bundled compiler
- Size packaging: [agent-sandbox image packaging](./2026-08-30-agent-sandbox-image-packaging.md)

## Verification

- Dockerfile build-time offline `go run main.go` as `agent`
- `.github/workflows/docker-agent-sandbox.yml` smoke: `GOTOOLCHAIN=local` + `--network=none go run`
- `go test ./pkg/exec/ -run TestRunEntrypoint`
- Contract documented in [docs/agent/agent-sandbox.md](../../../../docs/agent/agent-sandbox.md)
