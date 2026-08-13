# Agent Note: Chatagent chat feed style

Status: implemented

## Problem

The Chat transcript used DaisyUI `chat-bubble` chrome: teal user pills, bordered assistant cards, and tool/thinking blocks that read as stacked cards. That fights a long operator session and does not match the quieter process-log + prose grammar of the Chat page.

## Decision

Chat keeps the same four roles (user / thinking / tool / assistant). Trajectory stays the second reading for SYSTEM/CONTEXT/permission. The feed restyles those roles:

- User: muted `base-200` bubble (`--flowbot-radius-box`), copy plain text under the bubble. No Turn/Total on user rows.
- Thinking / tool: collapsed one-liner `[icon] label · first-line preview` with duration flush-right; the disclosure caret is reserved but **only painted when the row is open**. error / failed / needs_approval still auto-expand. Preview is first line then ~72 runes; tool prefers `Text`, else stdout. Generic tool icon (not a terminal for every tool). Expanded stdout/stderr is indented log text, not a card.
- Assistant: unboxed prose. Copy markdown + Turn/Total sit in a sibling meta row so streaming `innerHTML` cannot wipe them. Inner-loop `turn` SSE events update that meta; they do not insert a `Step N · duration` row in the feed.
- Markdown CSS is scoped to `.chatagent-assistant-body`; the streaming renderer passes those classes explicitly.

Composer, header, approval, todos, and Trajectory are unchanged. No thumbs, retry, first-token, or tok/s.

## Alternatives considered

- **Merge Trajectory process rows into Chat.** Rejected: duplicates the inspect view and would fabricate historical SYSTEM/CONTEXT.
- **Keep DaisyUI bubbles, only lower contrast.** Rejected: the screenshot grammar is log + prose, not quieter cards.
- **Independent tool title field.** Rejected: existing Text/stdout is enough for a truncated summary.

## Consequences

- SSR templates and `public/js/chatagent-thread.js` must stay isomorphic; `chatagent-markdown.js` must not restore `chat-bubble` classes on delta.
- Session list preview truncate stays separate (`strings.Fields` on the whole text); feed summaries take the first line first.
- Tool status is not painted in the collapsed one-liner; expand still keys off the status string (`data-tool-status` on `<details>`).
- User copy uses `data-clip-text`; assistant copy uses `data-clip-markdown`.

## Verification

- `pkg/views/partials`: user muted bubble + `data-clip-text`; assistant has no `chat-bubble` and meta as a sibling of the body; thinking/tool collapse and first-line preview; tool one-liner omits status; `ChatAgentOneLinePreview` / `ChatAgentToolPreview`.
- Web: Chat feed on `/service/web/agents/:id`; Trajectory toggle unchanged.
