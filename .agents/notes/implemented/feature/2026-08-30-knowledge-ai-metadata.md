# Agent Note: Knowledge form AI metadata generation

Status: implemented

## Problem

Operators writing Agent Knowledge markdown must hand-fill Title, Tags, and Summary for `search_knowledge` / `get_knowledge`. After drafting Content, those fields are repetitive and easy to leave empty or inconsistent with the body.

## Decision

The Knowledge edit form exposes **Generate metadata** as an outline primary button on a toolbar row above Title/Tags/Summary (hint on the left, action on the right) when `chat_agent.chat_model` is set. A click always overwrites those three fields from Path + Content via a single-shot LLM call (`chatagent.GenerateKnowledgeMetadata`), re-renders the HTMX form row, and does **not** persist until Save. Empty Content disables the button client-side and is rejected server-side with a toast. Unconfigured chat agent hides the button; runtime LLM failures toast without swapping the form. Metadata language follows Content; Content may be truncated for the prompt (oversized bodies are not rejected solely for generation). After sanitize, tags must be 3–6 short values (each ≤32 runes) or generation fails as incomplete.

## Alternatives considered

- **Fill only empty fields.** Rejected: explicit generate should rewrite metadata; partial fill looks like a failed run on edit.
- **Persist on generate.** Rejected: draft assist must stay undoable via Cancel; overwrite-plus-write is too sharp.
- **Separate `metadata_model` config.** Rejected: low-frequency ops action; reuse `chat_model` until cost forces a knob.
- **JSON + client JS field patch.** Rejected: existing Save already swaps the form row via HTMX.
- **Put LLM logic in `internal/modules/web`.** Rejected: product orchestration belongs with session title / knowledge path in `chatagent`.
- **Reject generate when Content exceeds the form byte cap.** Rejected: generation truncates for the prompt; Save still enforces the form size limit.

## Consequences

- Generate requires a configured ChatAgent model; the button is absent otherwise.
- Path is never AI-generated.
- Large Content is truncated for the prompt; summaries reflect the head of the document.
- Fewer than three tags after sanitize fails the generate request (toast), leaving the form unchanged.

## Verification

- `go test ./internal/server/chatagent -run KnowledgeMetadata` — prompt truncate, JSON parse/sanitize, min-tag reject, injected generator, FakeModel path.
- `go test ./internal/modules/web -run 'AgentKnowledge(NewFormShowsGenerate|GenerateMetadata)'` — button visibility; fill-without-persist; empty content and disabled-agent toasts.
