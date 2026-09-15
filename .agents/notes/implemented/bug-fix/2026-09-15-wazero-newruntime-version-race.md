# Agent Note: Serialize wazero NewRuntime (version cache race)

Status: implemented

## Problem

`go test -race ./pkg/plugin/wasm` fails when parallel tests call `NewWasmRunner` concurrently. The race is inside wazero v1.12.0 `internal/version.GetWazeroVersion`: a package-level `version` string is lazily written without synchronization. Parallel top-level tests (`TestNewWasmRunner`, `TestWasmRunnerCustomTimeout`, …) each create a Runtime and trip the detector. The same race would appear in production if multiple wasm plugins load at once.

## Decision

Wrap `wazero.NewRuntime` behind a package-level mutex in `pkg/plugin/wasm` (`newWazeroRuntime`). Plugin load is infrequent; serializing Runtime construction is acceptable until a wazero release ships the upstream fix (main already moves version detection into `init()`).

## Alternatives considered

- **Drop `t.Parallel()` from wasm runner tests only.** Rejected as the sole fix: hides CI failures but leaves production concurrent loads racing.
- **Upgrade wazero to an unreleased main commit.** Rejected: no tagged release yet; prefer a local workaround over pinning a floating commit.
- **Share one Runtime across WasmRunners.** Rejected: larger design change than this defect needs; runners already own their Runtime lifecycle.

## Consequences

- Concurrent `NewWasmRunner` calls no longer race through wazero’s version cache.
- Runtime creation is briefly serialized; negligible for plugin load paths.
- When wazero releases the `init()`-based version lookup, the mutex can be removed.

## Verification

- `go test -race ./pkg/plugin/wasm -count=1` (includes `TestNewWasmRunnerConcurrent`)
