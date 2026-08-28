# Agent Note: Domain error Error() includes Cause

Status: implemented

## Problem

`types.WrapError` stores the underlying failure in `Cause`, but `types.Error.Error()` returned only `Message`. Pipeline logs, step-run rows, and live progress events all use `err.Error()`, so operator surfaces showed `kanboard create task` with no JSON-RPC or provider text.

## Decision

`Error()` joins `Message` with `Cause` (same shape as `fmt.Errorf("%s: %w", …)`), and skips the cause when `Message` already ends with it. HTTP JSON bodies use `types.ClientMessage`, which returns domain `Message` only. `protocol.NewFailedResponse` / `clientSafeDomainMessage` still read `Message`; chatagent handlers funnel stringify through `chatAgentError` / `ClientMessage`.

## Alternatives considered

- **Log-only unwrap in `flog.Error`.** Pipeline DB rows, SSE progress, and `fmt.Errorf("%w")` wrappers would still hide `Cause`.
- **Put the cause into `Message` at each `WrapError` call site.** Duplicates wrapping, and HTTP would then leak provider text because `NewFailedResponse` reads `Message`.

## Consequences

Operator logs and pipeline run errors include the provider chain. Fiber's domain error handler stays client-safe because it uses `NewFailedResponse`. Chatagent JSON uses `ClientMessage`, so authenticated HTTP does not leak `Cause`.

## Verification

`TestError_Error_JoinsCause`, `TestWrapError`, and `TestClientMessage_OmitsCause` in `pkg/types/errors_test.go`. `TestNewFailedResponse` case `domain wrap does not leak cause text` in `pkg/types/protocol/action_test.go`.
