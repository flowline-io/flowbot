# Agent Note: Compaction prunes tool results first, then reuses the warm request prefix

Status: implemented

## Problem

Automatic compaction is the most expensive auxiliary call in a long session: it fires when the conversation is largest. Two independent mistakes made that call cost more than the work it saved.

Pressure always invoked an LLM summarizer. Oversized tool results are the usual bulk, and they can be reduced without a model. The existing prune flag only dropped old tool payloads from the summarizer *prompt*, so the next conversation request still sent the full results.

The summarizer then issued a separate request whose prefix shared nothing with the just-routed conversation: a dedicated summarizer system prompt plus a flattened transcript. Providers cache the leading token sequence. A different first token invalidates the whole prefix, so the largest turn paid full prompt processing twice.

Overflow recovery retried after any successful compact call, and joined a compaction failure onto the provider error. That could look repaired when the model-visible surface had not changed, or hide the original context-window rejection.

## Decision

Compaction stays inside `pkg/agent/ctxmgr`. Capacity remains the catalog window on `ctxmgr.Manager`; threshold, retain, prune, and retry stay on `ctxmgr.Settings`. Token estimates stay model-agnostic (`EstimateTokens`) and are not a second model registry.

### Layered reduction

Once pressure or overflow has already qualified, optional prune rewrites oversized current tool results to a bounded head, a fixed omission marker, and a bounded tail. Replacements are appended as session messages with the same `tool_call_id`; `session.SanitizeToolMessageOrder` and compaction overlay keep the latest payload in the original position. After prune, usage is remeasured. If pressure is gone, summarization is skipped.

Overflow (`CompactAndReload`) uses the same prune-first path. Force continues to a shrinking summary when a useful range exists. A prune-only pass that changes the surface is enough progress to retry without a summary.

### Summarization reuses the warm prefix

`RunCompaction` converts shadowed agent messages with `transform.DefaultConvertToLLM` after `session.SanitizeToolMessageOrder`, keeps the conversation system prompt and tool schemas, and appends the compaction instruction as the final user message. Tools ride along even though the summarizer must not call one. A prior checkpoint is prepended as `CompactionSummaryMessage` so the replay matches `session.BuildContext`. Persisted tool-call ids are already set; convert still fills any empty id.

Prefix reuse is the default. It is explicitly forgone when: the range is a non-head manual compact; the summarizer model differs from the conversation model; the system prompt is empty; the summarizer has no tools while the conversation did; or prune rewrote tool results in the same pass (the overflowed request's cached prefix still holds the unpruned payloads, so the summarizer must see the pruned surface to fit). Overlay still applies the pruned payloads onto the shadowed head before convert.

Branch summarization (`MoveTo`) uses the same trailing-instruction call shape but does not claim prefix reuse.

### Overflow retry requires surface progress

`CompactAndReload` returns `CompactReport`. The harness retries only when `Changed()` is true (prune and/or summary landed). Summary failure after a successful prune still retries. No progress keeps the original provider overflow error as the run result.

## Alternatives considered

- **Port the Cordis compaction / pruner / token-meter plugin split.** Rejected: flowbot has one implementation; a hypothetical seam is not worth three packages.
- **In-place Storage.Replace for pruned tool results.** Rejected for this change: append-only session storage stays the persistence model; last-write-wins overlay reconstructs the surface.
- **Keep the summarizer system prompt and flatten only the body.** Rejected: the system slot is the first cached token region; a distinct summarizer prompt still misses the entire prefix.
- **Omit tools from the summarization request because the model will not call one.** Rejected: tool schemas are part of the cached token sequence.
- **Retry overflow whenever CompactAndReload returns nil error.** Rejected: success without a surface change must not look repaired.

## Consequences

- Pressure can drop below threshold with no summarization call when tool results were the bulk.
- The summarization request is a prefix extension of the last routed conversation request when system, tools, and shadowed messages match.
- Overflow recovery is monotonic: retry only after the model-visible surface changes; the provider error remains authoritative otherwise.
- Dual-model window resolution is unchanged; `Manager` still uses its constructor `ContextWindow`.

## Verification

- `pkg/agent/ctxmgr`: prune rewrites oversized tool results and is idempotent; `EnsureWithinBudget` skips the LLM when prune relieves pressure; `RunCompaction` sanitizes then forwards conversation system, tools, thinking, converted messages, and a trailing instruction; `CompactAndReload` reports prune vs summary progress via `Changed()`.
- `pkg/agent/llm`: `CompleteWithTools` always passes `WithTools` and attaches thinking plus prior-turn reasoning; `FakeModel` records the last messages, tools, and context.
- `pkg/agent/harness`: overflow retries after prune-only progress; no retry when compaction does not change the surface; disabled compaction still returns the original overflow error.
- Owning map: [docs/agent/architecture.md](../../../../docs/agent/architecture.md) Context Management.
