# Agent Note: Wire reserved agent hooks

Status: implemented

## Problem

The typed hooks MVP shipped `before_agent_start`, `context`, `tool_call`, and `tool_result`, but left provider-request and session compact/tree events reserved. Product and engine extensions that need to patch LLM stream options, cancel/customize compaction, or customize branch summaries had no supported seam.

## Decision

Wire three events through `pkg/agent/hooks` with pi-aligned reduce rules:

- `before_provider_request` patches `msg.ProviderRequestOptions` (`Temperature` / `MaxTokens` / `ThinkingLevel` / retry) via `msg.Config.BeforeProviderRequest`, composed by `hooks.BridgeConfig`, applied in `loop.streamAssistant` after `buildStreamOptions` and before `llm.StreamAssistant`.
- `session_before_compact` and `session_before_tree` use first-cancel / last-custom-result reduce (`reduceFirstCancelOrLast`). Harness injects registry adapters into `ctxmgr.Manager` only when `HasSessionHandlers()` is true (register session hooks before `harness.New`); ctxmgr does not import hooks.
- Compaction call sites set typed `CompactOpts.Reason` (`manual` / `threshold` / `overflow`) and `WillRetry` on overflow recovery.
- `MoveTo` always emits `BeforeTree` on a real leaf change. `UserWantsSummary` is true when the caller left summary empty (auto-generate), false when the caller already supplied summary text.

`before_provider_payload` and `after_provider_response` stay unimplemented: langchaingo does not expose payload/response callbacks, and adding an HTTP body rewrite seam is a separate change.

## Alternatives considered

- Mutate raw HTTP JSON like `thinkingTransport`: rejected for this change; it couples hooks to OpenAI-compatible wire format and every transport stack.
- Import hooks from ctxmgr: rejected to keep ctxmgr free of the hooks package; harness owns the adapter.
- Always wire empty session adapters: rejected in favor of `HasSessionHandlers()` gating so product code must register before `harness.New` (same contract as chatagent `RegisterHooks`).
- Observe-only compact notifications without cancel/custom summary: rejected; pi’s mutable compact/tree hooks are the useful extension surface.

## Consequences

- Loop `HasLoopHandlers` includes `before_provider_request`; session hooks require registration before harness construction.
- Hook cancel on manual compact maps to a no-op result in `chatagent.CompactSession`; tree cancel is treated like aborted navigation in `harness.MoveTo`.
- Custom compaction/tree summaries skip the default summarization LLM call.
- Compaction cancel uses a dedicated `afterCompactCancelled` path distinct from summarization failure.

## Verification

- `go test ./pkg/agent/hooks/... ./pkg/agent/ctxmgr/... ./pkg/agent/loop/... ./pkg/agent/harness/... ./pkg/agent/llm/...`
- `go tool task lint`
