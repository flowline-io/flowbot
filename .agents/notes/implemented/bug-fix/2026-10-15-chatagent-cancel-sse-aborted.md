# Agent Note: Chatagent cancel SSE must treat ErrAborted as canceled

Status: implemented

## Problem

`POST /chatagent/sessions/:id/cancel` aborted the in-flight LLM call, but the messages SSE stream never emitted `type=canceled`. The BDD case `accepts cancel while a run is in flight` failed: `sawCanceled` stayed false. Logs showed `agent llm: aborted` and `cancelled in-flight run`, so cancel worked; the stream path misclassified the error.

`pkg/agent/llm` mapped a cancelled generate to bare `agentllm.ErrAborted` (it did not wrap `context.Canceled`). The loop returned that sentinel unchanged, and `StreamAPIRun` / `writeStreamEventsUntilRunDone` only treated `context.Canceled` as user cancel—so the client got an `error` event (or nothing matching `canceled`).

Dependabot bumps (e.g. prometheus/common) were unrelated; they only re-ran the suite.

## Decision

- LLM abort helpers wrap `ctx.Err()` under `agentllm.ErrAborted` so cancel vs deadline remains visible via `errors.Is`.
- Normalize abort errors to `loop.ErrAborted` in `abortLoopError` (preserving the cause chain) when `StreamAssistant` fails, and count that path as metric status `cancelled`.
- In chatagent, `isRunAbortError` / `isRunInterrupted` treat cancel and abort sentinels as stream terminals; `abortStreamMessage` selects `run canceled by user`, `run timed out`, or `run canceled`.

## Alternatives considered

- Only change the BDD assertion to accept `error` with aborted text: would hide the product contract that cancel emits `canceled` for observers and the web UI; rejected.
- Soften every abort message to a single `run canceled` string without preserving deadline: simpler, but UI/logs cannot tell timeout from user cancel; rejected once the cause chain was cheap to keep.

## Consequences

- Cooperative cancel emits SSE `canceled` with `run canceled by user` when `context.Canceled` is in the chain.
- Run timeout emits SSE `canceled` with `run timed out` when `context.DeadlineExceeded` is in the chain.
- Bare `ErrAborted` (no ctx cause) emits `run canceled`.

## Verification

```bash
go test ./pkg/agent/llm/ ./pkg/agent/loop/ ./internal/server/chatagent/ -count=1 -run "Abort|Canceled|IsRun|WriteStreamEvents|Deadline"
```
